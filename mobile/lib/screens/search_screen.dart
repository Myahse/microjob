import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../api/models.dart';
import '../state/session.dart';
import '../util/format.dart';
import '../widgets/common.dart';
import 'public_profile_screen.dart';

/// "Je cherche un plombier" — search workers by skill or name.
class SearchScreen extends StatefulWidget {
  const SearchScreen({super.key});

  @override
  State<SearchScreen> createState() => _SearchScreenState();
}

class _SearchScreenState extends State<SearchScreen> {
  final q = TextEditingController();
  String query = '';

  @override
  Widget build(BuildContext context) {
    final s = context.read<Session>();
    return Scaffold(
      appBar: AppBar(
        title: TextField(
          controller: q,
          autofocus: true,
          decoration: const InputDecoration(hintText: 'Plombier, technicien informatique…', border: InputBorder.none),
          onSubmitted: (v) => setState(() => query = v),
        ),
        actions: [IconButton(onPressed: () => setState(() => query = q.text), icon: const Icon(Icons.search))],
      ),
      body: AsyncList<WorkerResult>(
        key: ValueKey(query),
        load: () => s.api.searchWorkers(query, lat: s.user!.latitude, lng: s.user!.longitude),
        empty: 'Aucun prestataire trouvé.',
        itemBuilder: (context, w, _) => ListTile(
          leading: UserAvatar(url: w.avatarUrl, name: w.firstName),
          title: Text('${w.firstName} ${w.lastName} ${w.available ? '🟢' : '⚪'}'),
          subtitle: Text(
            [
              if (w.headline.isNotEmpty) w.headline,
              '⭐ ${w.rating.toStringAsFixed(1)} (${w.ratingCount})',
              if (w.distanceKm != null) '📍 ${km(w.distanceKm)}',
              if (w.hourlyRate > 0) '${fcfa(w.hourlyRate)}/h',
            ].join(' · '),
          ),
          trailing: OutlinedButton(
            onPressed: () => Navigator.push(context, MaterialPageRoute(builder: (_) => PublicProfileScreen(userId: w.id))),
            child: const Text('CONTACT'),
          ),
        ),
      ),
    );
  }
}
