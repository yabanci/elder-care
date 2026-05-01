// Combined messaging screens: thread list + per-thread chat. Used by both
// patient and care roles since the data shape is identical (caregivers
// for patient view, patients for care view).
import "package:flutter/material.dart";
import "package:flutter_riverpod/flutter_riverpod.dart";
import "package:go_router/go_router.dart";
import "package:intl/intl.dart";

import "../api/api_client.dart";
import "../l10n/strings.dart";
import "../models/models.dart";
import "../state/providers.dart";

class ThreadsScreen extends ConsumerStatefulWidget {
  const ThreadsScreen({super.key, this.bottomNav});
  final Widget? bottomNav;
  @override
  ConsumerState<ThreadsScreen> createState() => _ThreadsScreenState();
}

class _ThreadsScreenState extends ConsumerState<ThreadsScreen> {
  bool _loading = true;
  List<_Contact> _contacts = [];

  @override
  void initState() {
    super.initState();
    _refresh();
  }

  Future<void> _refresh() async {
    final api = ref.read(apiClientProvider);
    final user = ref.read(authProvider).user;
    if (user == null) return;
    setState(() => _loading = true);
    try {
      final raw = user.role == "patient"
          ? await api.get("/api/caregivers")
          : await api.get("/api/patients");
      setState(() {
        _contacts = (raw as List).map((e) {
          if (user.role == "patient") {
            final c = Caregiver.fromJson(e);
            return _Contact(
                id: c.id,
                name: c.fullName,
                relation: c.relation);
          }
          final p = LinkedPatient.fromJson(e);
          return _Contact(
              id: p.patientId,
              name: p.fullName,
              relation: tr("role_patient",
                  ref.read(langProvider)));
        }).toList();
        _loading = false;
      });
    } on ApiException {
      setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final lang = ref.watch(langProvider);
    final user = ref.watch(authProvider).user;
    return Scaffold(
      appBar: AppBar(title: Text(tr("messages_title", lang))),
      body: RefreshIndicator(
        onRefresh: _refresh,
        child: _loading
            ? const Center(child: CircularProgressIndicator())
            : _contacts.isEmpty
                ? ListView(children: [
                    const SizedBox(height: 80),
                    Center(child: Text(tr("messages_no_threads", lang))),
                  ])
                : ListView.builder(
                    padding: const EdgeInsets.all(16),
                    itemCount: _contacts.length,
                    itemBuilder: (ctx, i) {
                      final c = _contacts[i];
                      return Card(
                        child: ListTile(
                          leading: CircleAvatar(child: Text(_initials(c.name))),
                          title: Text(c.name),
                          subtitle: Text(c.relation),
                          trailing: const Icon(Icons.chevron_right),
                          onTap: () {
                            final base =
                                user!.role == "patient" ? "/patient" : "/care";
                            context.push("$base/messages/${c.id}");
                          },
                        ),
                      );
                    }),
      ),
      bottomNavigationBar: widget.bottomNav,
    );
  }
}

class _Contact {
  _Contact({required this.id, required this.name, required this.relation});
  final String id;
  final String name;
  final String relation;
}

String _initials(String name) {
  final parts = name.split(" ").where((p) => p.isNotEmpty).toList();
  return parts.take(2).map((p) => p[0].toUpperCase()).join();
}

class ThreadScreen extends ConsumerStatefulWidget {
  const ThreadScreen({super.key, required this.otherId});
  final String otherId;
  @override
  ConsumerState<ThreadScreen> createState() => _ThreadScreenState();
}

class _ThreadScreenState extends ConsumerState<ThreadScreen> {
  final _controller = TextEditingController();
  final _scroll = ScrollController();
  bool _loading = true;
  List<Message> _messages = [];

  @override
  void initState() {
    super.initState();
    _refresh();
  }

  Future<void> _refresh() async {
    setState(() => _loading = true);
    try {
      final raw = await ref
          .read(apiClientProvider)
          .get("/api/messages/${widget.otherId}");
      setState(() {
        _messages = (raw as List).map((e) => Message.fromJson(e)).toList();
        _loading = false;
      });
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (_scroll.hasClients) {
          _scroll.jumpTo(_scroll.position.maxScrollExtent);
        }
      });
    } on ApiException {
      setState(() => _loading = false);
    }
  }

  Future<void> _send() async {
    final body = _controller.text.trim();
    if (body.isEmpty) return;
    _controller.clear();
    try {
      await ref.read(apiClientProvider).post("/api/messages",
          data: {"recipient_id": widget.otherId, "body": body});
      await _refresh();
    } catch (_) {}
  }

  @override
  Widget build(BuildContext context) {
    final lang = ref.watch(langProvider);
    final me = ref.watch(authProvider).user;
    return Scaffold(
      appBar: AppBar(
          title: Text(tr("messages_title", lang)),
          leading: BackButton(onPressed: () => context.pop())),
      body: Column(children: [
        Expanded(
          child: _loading
              ? const Center(child: CircularProgressIndicator())
              : ListView.builder(
                  controller: _scroll,
                  padding: const EdgeInsets.all(16),
                  itemCount: _messages.length,
                  itemBuilder: (ctx, i) {
                    final m = _messages[i];
                    final mine = me != null && m.senderId == me.id;
                    return Align(
                      alignment: mine
                          ? Alignment.centerRight
                          : Alignment.centerLeft,
                      child: Container(
                        margin: const EdgeInsets.symmetric(vertical: 4),
                        padding: const EdgeInsets.all(10),
                        decoration: BoxDecoration(
                          color: mine
                              ? Theme.of(context).colorScheme.primary
                              : Colors.grey.shade200,
                          borderRadius: BorderRadius.circular(12),
                        ),
                        constraints: BoxConstraints(
                          maxWidth: MediaQuery.of(context).size.width * 0.75,
                        ),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(m.body,
                                style: TextStyle(
                                    color:
                                        mine ? Colors.white : Colors.black)),
                            const SizedBox(height: 4),
                            Text(
                              DateFormat.Hm().format(m.createdAt),
                              style: TextStyle(
                                color: mine ? Colors.white70 : Colors.black54,
                                fontSize: 11,
                              ),
                            ),
                          ],
                        ),
                      ),
                    );
                  },
                ),
        ),
        SafeArea(
          top: false,
          child: Padding(
            padding: const EdgeInsets.all(8),
            child: Row(children: [
              Expanded(
                child: TextField(
                  controller: _controller,
                  decoration: InputDecoration(
                      hintText: tr("message_placeholder", lang)),
                  onSubmitted: (_) => _send(),
                ),
              ),
              IconButton(
                  icon: const Icon(Icons.send), onPressed: _send),
            ]),
          ),
        ),
      ]),
    );
  }
}
