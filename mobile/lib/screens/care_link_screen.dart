import "package:flutter/material.dart";
import "package:flutter_riverpod/flutter_riverpod.dart";
import "package:go_router/go_router.dart";

import "../api/api_client.dart";
import "../l10n/strings.dart";
import "../state/providers.dart";

class CareLinkScreen extends ConsumerStatefulWidget {
  const CareLinkScreen({super.key});
  @override
  ConsumerState<CareLinkScreen> createState() => _CareLinkScreenState();
}

class _CareLinkScreenState extends ConsumerState<CareLinkScreen> {
  final _code = TextEditingController();
  bool _busy = false;
  String? _error;
  String? _success;

  Future<void> _submit() async {
    setState(() {
      _busy = true;
      _error = null;
      _success = null;
    });
    try {
      await ref.read(apiClientProvider).post("/api/patients/link",
          data: {"invite_code": _code.text.trim().toUpperCase()});
      setState(() => _success = tr("care_link_success",
          ref.read(langProvider)));
      _code.clear();
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
      appBar: AppBar(title: Text(tr("care_link_title", lang))),
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(
                tr("invite_code_hint", lang),
                style: Theme.of(context).textTheme.bodyMedium,
              ),
              const SizedBox(height: 16),
              TextField(
                controller: _code,
                autofocus: true,
                textCapitalization: TextCapitalization.characters,
                decoration: InputDecoration(
                    labelText: tr("care_link_code", lang),
                    hintText: "ELDER001"),
              ),
              if (_error != null) ...[
                const SizedBox(height: 12),
                Text(_error!,
                    style: TextStyle(
                        color: Theme.of(context).colorScheme.error)),
              ],
              if (_success != null) ...[
                const SizedBox(height: 12),
                Text(_success!,
                    style: const TextStyle(color: Colors.green)),
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
                    : Text(tr("care_link_submit", lang)),
              ),
              const SizedBox(height: 12),
              if (_success != null)
                OutlinedButton(
                  onPressed: () => context.pop(),
                  child: Text(tr("back", lang)),
                ),
            ],
          ),
        ),
      ),
    );
  }
}
