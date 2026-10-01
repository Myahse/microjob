import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../api/models.dart';
import '../state/session.dart';
import '../util/format.dart';
import '../widgets/common.dart';
import 'chat_screen.dart';

class ConversationsScreen extends StatelessWidget {
  const ConversationsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final api = context.read<Session>().api;
    return AsyncList<Conversation>(
      load: api.conversations,
      reloadOn: (e) => e.type == 'message' || e.type == 'job',
      empty: "Une conversation s'ouvre automatiquement quand un job est attribué.",
      itemBuilder: (context, c, reload) => ListTile(
        leading: UserAvatar(url: c.other.avatarUrl, name: c.other.firstName),
        title: Text(c.other.fullName),
        subtitle: Text('${c.jobTitle}\n${c.lastMessage ?? ''}', maxLines: 2, overflow: TextOverflow.ellipsis),
        isThreeLine: true,
        trailing: c.lastAt == null ? null : Text(ago(c.lastAt!), style: const TextStyle(fontSize: 11)),
        onTap: () async {
          await Navigator.push(
            context,
            MaterialPageRoute(
              builder: (_) => ChatScreen(jobId: c.jobId, title: c.jobTitle, closed: c.closed || c.jobStatus == 'CANCELLED'),
            ),
          );
          reload();
        },
      ),
    );
  }
}
