// Riverpod providers wiring the API client + global state (current user,
// language). Kept minimal: only state that genuinely outlives a single
// widget tree lives here; per-screen state stays in the widget itself.
import "dart:async";
import "dart:convert";

import "package:flutter/foundation.dart";
import "package:flutter_riverpod/flutter_riverpod.dart";
import "package:shared_preferences/shared_preferences.dart";

import "../api/api_client.dart";
import "../models/models.dart";

/// Resolves the API base URL from a compile-time --dart-define so a
/// single Flutter binary can target dev / staging / prod without
/// rebuilding logic. Defaults to the Android-emulator host alias since
/// that's the most common dev target.
const apiBaseUrl =
    String.fromEnvironment("ELDERCARE_API", defaultValue: "http://10.0.2.2:8090");

final apiClientProvider = Provider<ApiClient>((ref) {
  return ApiClient(baseUrl: apiBaseUrl);
});

/// Auth state: null while loading or signed-out, populated when /me
/// succeeds. UI redirects from splash based on this.
class AuthState {
  AuthState({this.user, this.loading = true});
  final User? user;
  final bool loading;

  AuthState copyWith({User? user, bool? loading, bool clearUser = false}) =>
      AuthState(
        user: clearUser ? null : (user ?? this.user),
        loading: loading ?? this.loading,
      );
}

class AuthNotifier extends StateNotifier<AuthState> {
  AuthNotifier(this._api) : super(AuthState()) {
    _bootstrap();
  }
  final ApiClient _api;

  Future<void> _bootstrap() async {
    try {
      final raw = await _api.get("/api/me");
      state = AuthState(user: User.fromJson(raw as Map<String, dynamic>),
          loading: false);
      await _cacheUser(state.user!);
    } catch (e) {
      state = AuthState(loading: false);
    }
  }

  Future<void> login(String email, String password) async {
    final raw = await _api.post("/api/auth/login",
        data: {"email": email, "password": password});
    final m = raw as Map<String, dynamic>;
    await _api.saveToken(m["token"] as String);
    final u = User.fromJson(m["user"] as Map<String, dynamic>);
    state = AuthState(user: u, loading: false);
    await _cacheUser(u);
  }

  Future<void> register({
    required String email,
    required String password,
    required String fullName,
    required String role,
    String? phone,
    String? birthDate,
  }) async {
    final raw = await _api.post("/api/auth/register", data: {
      "email": email,
      "password": password,
      "full_name": fullName,
      "role": role,
      if (phone != null && phone.isNotEmpty) "phone": phone,
      if (birthDate != null && birthDate.isNotEmpty) "birth_date": birthDate,
    });
    final m = raw as Map<String, dynamic>;
    await _api.saveToken(m["token"] as String);
    final u = User.fromJson(m["user"] as Map<String, dynamic>);
    state = AuthState(user: u, loading: false);
    await _cacheUser(u);
  }

  Future<void> logout() async {
    try {
      await _api.post("/api/auth/logout");
    } catch (_) {
      // best-effort; we still clear local state below
    }
    await _api.clearToken();
    state = AuthState(loading: false); // user defaults to null
  }

  Future<void> updateProfile(Map<String, dynamic> patch) async {
    final raw = await _api.patch("/api/me", data: patch);
    final u = User.fromJson(raw as Map<String, dynamic>);
    state = state.copyWith(user: u);
    await _cacheUser(u);
  }

  Future<void> _cacheUser(User u) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString("user_blob", jsonEncode(u.toJson()));
  }
}

final authProvider = StateNotifierProvider<AuthNotifier, AuthState>((ref) {
  return AuthNotifier(ref.watch(apiClientProvider));
});

/// Language selection — stored locally; synced to backend on change.
class LangNotifier extends StateNotifier<String> {
  LangNotifier() : super("ru") {
    _load();
  }

  Future<void> _load() async {
    final prefs = await SharedPreferences.getInstance();
    final saved = prefs.getString("lang");
    if (saved != null && const ["ru", "kk", "en"].contains(saved)) {
      state = saved;
    } else {
      // Fall back to OS locale.
      final code = PlatformDispatcher.instance.locale.languageCode;
      if (code == "kk" || code == "en") state = code;
    }
  }

  Future<void> set(String code) async {
    if (state == code) return;
    state = code;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString("lang", code);
  }
}

final langProvider =
    StateNotifierProvider<LangNotifier, String>((ref) => LangNotifier());
