import "package:flutter/material.dart";
import "package:flutter_riverpod/flutter_riverpod.dart";
import "package:go_router/go_router.dart";

import "../api/api_client.dart";
import "../l10n/strings.dart";
import "../state/providers.dart";
import "../widgets/lang_switcher.dart";

class RegisterScreen extends ConsumerStatefulWidget {
  const RegisterScreen({super.key});
  @override
  ConsumerState<RegisterScreen> createState() => _RegisterScreenState();
}

class _RegisterScreenState extends ConsumerState<RegisterScreen> {
  final _email = TextEditingController();
  final _password = TextEditingController();
  final _fullName = TextEditingController();
  final _phone = TextEditingController();
  final _birthDate = TextEditingController();
  String _role = "patient";
  bool _busy = false;
  String? _error;

  Future<void> _submit() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await ref.read(authProvider.notifier).register(
            email: _email.text.trim(),
            password: _password.text,
            fullName: _fullName.text.trim(),
            role: _role,
            phone: _phone.text.trim(),
            birthDate: _birthDate.text.trim(),
          );
      if (!mounted) return;
      final user = ref.read(authProvider).user!;
      if (user.role == "patient") {
        context.go("/patient/onboarding");
      } else {
        context.go("/care");
      }
    } on ApiException catch (e) {
      setState(() => _error = e.message);
    } catch (e) {
      setState(() => _error = e.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final lang = ref.watch(langProvider);
    return Scaffold(
      appBar: AppBar(
        title: Text(tr("register_title", lang)),
        actions: const [LangSwitcher(), SizedBox(width: 8)],
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              SegmentedButton<String>(
                segments: [
                  ButtonSegment(
                    value: "patient",
                    label: Text(tr("role_patient", lang)),
                  ),
                  ButtonSegment(
                    value: "doctor",
                    label: Text(tr("role_doctor", lang)),
                  ),
                  ButtonSegment(
                    value: "family",
                    label: Text(tr("role_family", lang)),
                  ),
                ],
                selected: {_role},
                onSelectionChanged: (s) => setState(() => _role = s.first),
              ),
              const SizedBox(height: 16),
              TextField(
                controller: _email,
                keyboardType: TextInputType.emailAddress,
                decoration: InputDecoration(labelText: tr("login_email", lang)),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _password,
                obscureText: true,
                decoration:
                    InputDecoration(labelText: tr("login_password", lang)),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _fullName,
                decoration:
                    InputDecoration(labelText: tr("register_fullname", lang)),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _phone,
                keyboardType: TextInputType.phone,
                decoration:
                    InputDecoration(labelText: tr("register_phone", lang)),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _birthDate,
                decoration:
                    InputDecoration(labelText: tr("register_birth", lang)),
              ),
              if (_error != null) ...[
                const SizedBox(height: 12),
                Text(_error!,
                    style: TextStyle(
                        color: Theme.of(context).colorScheme.error)),
              ],
              const SizedBox(height: 24),
              ElevatedButton(
                onPressed: _busy ? null : _submit,
                child: _busy
                    ? const SizedBox(
                        width: 18,
                        height: 18,
                        child:
                            CircularProgressIndicator(strokeWidth: 2),
                      )
                    : Text(tr("register_submit", lang)),
              ),
              const SizedBox(height: 12),
              TextButton(
                onPressed: () => context.go("/login"),
                child: Text(tr("register_have_account", lang)),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
