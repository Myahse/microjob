import 'dart:async';

import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';

import '../api/models.dart';
import '../state/session.dart';
import '../util/format.dart';

/// Job conversation, created automatically once a worker is assigned.
class ChatScreen extends StatefulWidget {
  const ChatScreen({super.key, required this.jobId, required this.title, this.closed = false});
  final String jobId;
  final String title;
  final bool closed;

  @override
  State<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends State<ChatScreen> {
  final text = TextEditingController();
  final scroll = ScrollController();
  List<Message> messages = [];
  StreamSubscription? sub;

  @override
  void initState() {
    super.initState();
    final s = context.read<Session>();
    load();
    sub = s.events.listen((e) {
      if (e.type == 'message' && e.data['job_id'] == widget.jobId) {
        final m = Message.fromJson(e.data);
        if (!messages.any((x) => x.id == m.id)) {
          setState(() => messages.add(m));
          _scrollDown();
        }
      } else if (e.type == 'job' && e.data['job_id'] == widget.jobId) {
        load();
      }
    });
  }

  @override
  void dispose() {
    sub?.cancel();
    super.dispose();
  }

  Future<void> load() async {
    try {
      final m = await context.read<Session>().api.messages(widget.jobId);
      if (mounted) setState(() => messages = m);
      _scrollDown();
    } catch (e) {
      if (mounted) toast(context, e);
    }
  }

  void _scrollDown() => WidgetsBinding.instance.addPostFrameCallback((_) {
    if (scroll.hasClients) scroll.jumpTo(scroll.position.maxScrollExtent);
  });

  Future<void> send(Map<String, dynamic> body) async {
    try {
      final m = await context.read<Session>().api.sendMessage(widget.jobId, body);
      if (!messages.any((x) => x.id == m.id)) setState(() => messages.add(m));
      _scrollDown();
    } catch (e) {
      if (mounted) toast(context, e);
    }
  }

  Future<void> sendPhoto() async {
    final api = context.read<Session>().api;
    final img = await ImagePicker().pickImage(source: ImageSource.camera, maxWidth: 1600, imageQuality: 80);
    if (img == null) return;
    try {
      final url = await api.upload(await img.readAsBytes(), img.name);
      await send({'kind': 'image', 'attachment_url': url});
    } catch (e) {
      if (mounted) toast(context, e);
    }
  }

  Future<void> sendLocation() async {
    try {
      final p = await currentPosition();
      await send({'kind': 'location', 'latitude': p.latitude, 'longitude': p.longitude});
    } catch (e) {
      if (mounted) toast(context, e);
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = context.read<Session>();
    final me = s.user!.id;
    return Scaffold(
      appBar: AppBar(title: Text(widget.title)),
      body: Column(
        children: [
          Expanded(
            child: ListView.builder(
              controller: scroll,
              padding: const EdgeInsets.all(12),
              itemCount: messages.length,
              itemBuilder: (context, i) {
                final m = messages[i];
                if (m.kind == 'system') {
                  return Center(
                    child: Container(
                      margin: const EdgeInsets.symmetric(vertical: 6),
                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                      decoration: BoxDecoration(color: Colors.grey.shade200, borderRadius: BorderRadius.circular(8)),
                      child: Text(m.body, textAlign: TextAlign.center, style: const TextStyle(fontSize: 12)),
                    ),
                  );
                }
                final mine = m.senderId == me;
                final fg = mine ? Colors.white : Colors.black87;
                return Align(
                  alignment: mine ? Alignment.centerRight : Alignment.centerLeft,
                  child: Container(
                    constraints: BoxConstraints(maxWidth: MediaQuery.of(context).size.width * 0.75),
                    margin: const EdgeInsets.symmetric(vertical: 3),
                    padding: const EdgeInsets.all(10),
                    decoration: BoxDecoration(
                      color: mine ? Colors.indigo : Colors.grey.shade100,
                      borderRadius: BorderRadius.circular(16),
                    ),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        if (m.kind == 'image')
                          ClipRRect(
                            borderRadius: BorderRadius.circular(8),
                            child: Image.network(s.api.media(m.attachmentUrl), height: 180),
                          ),
                        if (m.kind == 'location')
                          InkWell(
                            onTap: () => launchUrl(
                              Uri.parse('https://www.google.com/maps/search/?api=1&query=${m.latitude},${m.longitude}'),
                            ),
                            child: Text(
                              '📍 Voir la position',
                              style: TextStyle(color: fg, decoration: TextDecoration.underline),
                            ),
                          ),
                        if (m.body.isNotEmpty) Text(m.body, style: TextStyle(color: fg)),
                        Text(
                          '${m.createdAt.hour.toString().padLeft(2, '0')}:${m.createdAt.minute.toString().padLeft(2, '0')}',
                          style: TextStyle(fontSize: 10, color: mine ? Colors.white70 : Colors.grey),
                        ),
                      ],
                    ),
                  ),
                );
              },
            ),
          ),
          if (widget.closed)
            const Padding(padding: EdgeInsets.all(12), child: Text('Conversation fermée'))
          else
            SafeArea(
              child: Row(
                children: [
                  IconButton(onPressed: sendPhoto, icon: const Icon(Icons.photo_camera_outlined)),
                  IconButton(onPressed: sendLocation, icon: const Icon(Icons.place_outlined)),
                  Expanded(
                    child: TextField(
                      controller: text,
                      decoration: const InputDecoration(hintText: 'Votre message…', border: InputBorder.none),
                      onSubmitted: (_) => _sendText(),
                    ),
                  ),
                  IconButton(onPressed: _sendText, icon: const Icon(Icons.send)),
                ],
              ),
            ),
        ],
      ),
    );
  }

  void _sendText() {
    final t = text.text.trim();
    if (t.isEmpty) return;
    text.clear();
    send({'kind': 'text', 'body': t});
  }
}
