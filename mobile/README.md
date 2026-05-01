# ElderCare Mobile (Flutter)

Полнофункциональный клиент для бэкенда ElderCare. Замещает удалённый
Next.js фронт. Один кодбейс под Android, iOS и web.

## Стек

- Flutter 3.41+ / Dart 3.11
- Material 3 (адаптивная тема под пожилых: крупные шрифты, высокий контраст)
- GoRouter — декларативный роутинг
- Riverpod — state management
- Dio — HTTP client с Bearer-interceptor (читает токен из `flutter_secure_storage`)
- fl_chart — графики метрик
- intl — форматирование дат под локаль
- shared_preferences + flutter_secure_storage — language pref + JWT

## Запуск

```bash
make install      # из корня репо
make backend &    # backend на :8090
make mobile       # flutter run на текущем устройстве (chrome/emulator)
```

API base URL по умолчанию — `http://10.0.2.2:8090` (Android-эмулятор → host).
Для других целей передайте `--dart-define=ELDERCARE_API=...`:

```bash
flutter run --dart-define=ELDERCARE_API=http://192.168.1.10:8090
```

Web-target для быстрого preview:

```bash
flutter run -d chrome --dart-define=ELDERCARE_API=http://localhost:8090
```

## Тесты

```bash
make mobile-test   # flutter test
make lint          # flutter analyze + go vet
```

## Структура

```
mobile/
  lib/
    main.dart                 entrypoint + router + live-alert listener
    theme.dart                Material 3 theme (large-text accessibility)
    api/api_client.dart       Dio + Bearer token + SSE stream parser
    models/models.dart        DTOs mirroring backend JSON shapes
    state/
      providers.dart          authProvider, langProvider, apiClientProvider
      live_alerts.dart        SSE subscription → SnackBar push
    l10n/strings.dart         ru/kk/en string bundle (hand-rolled)
    widgets/
      lang_switcher.dart
      metric_meta.dart        per-metric emoji/unit/colour
    screens/
      splash_screen.dart      auth bootstrap → role-based redirect
      login_screen.dart, register_screen.dart
      patient_*.dart          home / metrics / meds / plans / alerts / messages / profile / onboarding
      care_*.dart             home / link / patient_detail / messages / profile
      messages_screen.dart    threads + thread (shared)
  test/widget_test.dart       smoke tests (login render + i18n coverage)
```

## Что не делает мобилка (сознательно)

- **Bluetooth с реальными устройствами**: список IoT-устройств в профиле
  был только UI-моком и в мобилке не повторяется. Если будет интеграция
  с настоящим тонометром / часами — отдельный пакет `lib/devices/`.
- **Web Push**: на мобилке используется Server-Sent Events (`/api/events`)
  для in-app live-alert. Для нативных push-уведомлений нужен FCM/APNs —
  отдельный шаг с настройкой Firebase project.

Backlog в `docs/superpowers/AUDIT.md` §9.
