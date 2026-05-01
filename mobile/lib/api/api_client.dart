// HTTP client + auth-token plumbing for the ElderCare backend.
//
// The Go backend exposes the same JSON API the deleted Next frontend
// used; we authenticate with Bearer tokens (mobile cannot store
// HttpOnly cookies). The token lives in flutter_secure_storage so the
// OS keychain protects it across cold starts.
import "dart:async";
import "dart:convert";

import "package:dio/dio.dart";
import "package:flutter_secure_storage/flutter_secure_storage.dart";
import "package:shared_preferences/shared_preferences.dart";

const _tokenKey = "eldercare_token";
const _apiBaseDefault = "http://10.0.2.2:8090"; // Android emulator → host

class ApiException implements Exception {
  ApiException(this.statusCode, this.message);
  final int statusCode;
  final String message;
  @override
  String toString() => "ApiException($statusCode): $message";
}

class ApiClient {
  ApiClient({String? baseUrl, FlutterSecureStorage? storage})
      : _storage = storage ?? const FlutterSecureStorage(),
        _dio = Dio(BaseOptions(
          baseUrl: baseUrl ?? _apiBaseDefault,
          connectTimeout: const Duration(seconds: 10),
          receiveTimeout: const Duration(seconds: 30),
          contentType: "application/json",
          // Don't auto-throw on non-2xx — the interceptor below normalises
          // all responses to ApiException so call sites have one error type.
          validateStatus: (_) => true,
        )) {
    _dio.interceptors.add(InterceptorsWrapper(
      onRequest: (options, handler) async {
        final token = await _storage.read(key: _tokenKey);
        if (token != null) {
          options.headers["Authorization"] = "Bearer $token";
        }
        handler.next(options);
      },
    ));
  }

  final Dio _dio;
  final FlutterSecureStorage _storage;

  /// Override the API base URL at runtime (used by widget tests).
  void setBaseUrl(String url) => _dio.options.baseUrl = url;

  Future<String?> readToken() => _storage.read(key: _tokenKey);

  Future<void> saveToken(String token) =>
      _storage.write(key: _tokenKey, value: token);

  Future<void> clearToken() async {
    await _storage.delete(key: _tokenKey);
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove("user_blob");
  }

  Future<dynamic> get(String path,
      {Map<String, dynamic>? queryParameters}) async {
    final res = await _dio.get(path, queryParameters: queryParameters);
    return _unwrap(res);
  }

  Future<dynamic> post(String path, {Object? data}) async {
    final res = await _dio.post(path, data: data);
    return _unwrap(res);
  }

  Future<dynamic> patch(String path, {Object? data}) async {
    final res = await _dio.patch(path, data: data);
    return _unwrap(res);
  }

  Future<dynamic> delete(String path) async {
    final res = await _dio.delete(path);
    return _unwrap(res);
  }

  /// Long-lived stream connection for SSE (`text/event-stream`). The
  /// returned stream emits one event per `data:` block. Closes when the
  /// caller cancels the subscription or the server hangs up.
  Stream<Map<String, dynamic>> sseStream(String path) async* {
    final response = await _dio.get<ResponseBody>(
      path,
      options: Options(
        responseType: ResponseType.stream,
        headers: {"Accept": "text/event-stream"},
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(response.statusCode ?? 0, "SSE handshake failed");
    }
    final body = response.data;
    if (body == null) return;
    final buffer = StringBuffer();
    await for (final chunk in body.stream) {
      buffer.write(utf8.decode(chunk, allowMalformed: true));
      while (true) {
        final s = buffer.toString();
        final boundary = s.indexOf("\n\n");
        if (boundary == -1) break;
        final raw = s.substring(0, boundary);
        buffer
          ..clear()
          ..write(s.substring(boundary + 2));
        final dataLine = raw.split("\n").firstWhere(
              (l) => l.startsWith("data:"),
              orElse: () => "",
            );
        if (dataLine.isEmpty) continue;
        final payload = dataLine.substring(5).trim();
        try {
          final decoded = jsonDecode(payload);
          if (decoded is Map<String, dynamic>) yield decoded;
        } catch (_) {
          // skip malformed events; SSE is best-effort
        }
      }
    }
  }

  dynamic _unwrap(Response res) {
    final code = res.statusCode ?? 0;
    if (code >= 200 && code < 300) return res.data;
    final msg = (res.data is Map && (res.data as Map)["error"] is String)
        ? (res.data as Map)["error"] as String
        : "HTTP $code";
    throw ApiException(code, msg);
  }
}
