import "package:flutter/material.dart";
import "package:flutter_riverpod/flutter_riverpod.dart";
import "package:go_router/go_router.dart";

import "../api/api_client.dart";
import "../l10n/strings.dart";
import "../state/providers.dart";

class PatientOnboardingScreen extends ConsumerStatefulWidget {
  const PatientOnboardingScreen({super.key});
  @override
  ConsumerState<PatientOnboardingScreen> createState() =>
      _PatientOnboardingScreenState();
}

class _PatientOnboardingScreenState
    extends ConsumerState<PatientOnboardingScreen> {
  final _height = TextEditingController();
  final _weight = TextEditingController();
  final _chronic = TextEditingController();
  final _bp = TextEditingController();
  final _meds = TextEditingController();
  bool _busy = false;
  String? _error;

  Future<void> _submit() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    final api = ref.read(apiClientProvider);
    try {
      final patch = <String, dynamic>{
        "onboarded": true,
        if (_height.text.trim().isNotEmpty)
          "height_cm": int.tryParse(_height.text.trim()),
        if (_chronic.text.trim().isNotEmpty)
          "chronic_conditions": _chronic.text.trim(),
        if (_bp.text.trim().isNotEmpty) "bp_norm": _bp.text.trim(),
        if (_meds.text.trim().isNotEmpty)
          "prescribed_meds": _meds.text.trim(),
      };
      await ref.read(authProvider.notifier).updateProfile(patch);

      // First weight reading goes into health_metrics so the BMI card
      // can render immediately.
      final w = double.tryParse(_weight.text.trim().replaceAll(",", "."));
      if (w != null && w > 0) {
        await api.post("/api/metrics", data: {"kind": "weight", "value": w});
      }

      if (!mounted) return;
      context.go("/patient");
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
          title: Text(tr("onboard_title", lang)), automaticallyImplyLeading: false),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(tr("onboard_sub", lang),
                  style: Theme.of(context).textTheme.bodyMedium),
              const SizedBox(height: 16),
              TextField(
                controller: _height,
                keyboardType: TextInputType.number,
                decoration: InputDecoration(labelText: tr("onboard_height", lang)),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _weight,
                keyboardType:
                    const TextInputType.numberWithOptions(decimal: true),
                decoration: InputDecoration(labelText: tr("onboard_weight", lang)),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _chronic,
                maxLines: 2,
                decoration: InputDecoration(
                    labelText: tr("onboard_chronic", lang),
                    hintText: "артериальная гипертензия, сахарный диабет 2 типа"),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _bp,
                decoration: InputDecoration(labelText: tr("onboard_bp_norm", lang)),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _meds,
                maxLines: 2,
                decoration:
                    InputDecoration(labelText: tr("onboard_meds", lang)),
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
                    : Text(tr("onboard_submit", lang)),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
