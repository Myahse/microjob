import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../api/models.dart';
import '../state/session.dart';
import '../util/format.dart';
import '../widgets/common.dart';
import 'job_detail_screen.dart';

class NotificationsScreen extends StatefulWidget {
  const NotificationsScreen({super.key});

  @override
  State<NotificationsScreen> createState() => _NotificationsScreenState();
}

class _NotificationsScreenState extends State<NotificationsScreen> {
  @override
  void initState() {
    super.initState();
    final s = context.read<Session>();
    s.api.markAllRead().then((_) => s.clearUnread()).catchError((_) {});
  }

  @override
  Widget build(BuildContext context) {
    final api = context.read<Session>().api;
    return Scaffold(
      appBar: AppBar(title: const Text('Notifications')),
      body: AsyncList<AppNotification>(
        load: () async => (await api.notifications()).$1,
        reloadOn: (e) => e.type == 'notification',
        empty: 'Aucune notification.',
        itemBuilder: (context, n, _) => ListTile(
          tileColor: n.read ? null : Colors.indigo.shade50,
          title: Text(n.title, style: const TextStyle(fontWeight: FontWeight.w600)),
          subtitle: n.body.isEmpty ? null : Text(n.body),
          trailing: Text(ago(n.createdAt), style: const TextStyle(fontSize: 11)),
          onTap: n.jobId == null
              ? null
              : () => Navigator.push(context, MaterialPageRoute(builder: (_) => JobDetailScreen(jobId: n.jobId!))),
        ),
      ),
    );
  }
}
