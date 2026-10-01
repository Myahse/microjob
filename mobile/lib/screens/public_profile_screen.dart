import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../api/models.dart';
import '../state/session.dart';
import '../util/format.dart';
import '../widgets/common.dart';

class PublicProfileScreen extends StatefulWidget {
  const PublicProfileScreen({super.key, required this.userId});
  final String userId;

  @override
  State<PublicProfileScreen> createState() => _PublicProfileScreenState();
}

class _PublicProfileScreenState extends State<PublicProfileScreen> {
  (User, List<Review>)? data;
  String? error;

  @override
  void initState() {
    super.initState();
    context
        .read<Session>()
        .api
        .profile(widget.userId)
        .then(
          (d) => mounted ? setState(() => data = d) : null,
          onError: (Object e) => mounted ? setState(() => error = e.toString()) : null,
        );
  }

  @override
  Widget build(BuildContext context) {
    final d = data;
    return Scaffold(
      appBar: AppBar(title: const Text('Profil')),
      body: d == null
          ? Center(child: error != null ? Text(error!) : const CircularProgressIndicator())
          : ListView(padding: const EdgeInsets.all(16), children: _content(d.$1, d.$2)),
    );
  }

  List<Widget> _content(User u, List<Review> reviews) {
    final w = u.worker;
    return [
      Center(
        child: UserAvatar(url: u.avatarUrl, name: u.firstName, radius: 44),
      ),
      const SizedBox(height: 8),
      Center(
        child: Text(u.fullName, style: const TextStyle(fontSize: 22, fontWeight: FontWeight.bold)),
      ),
      if (w != null && w.headline.isNotEmpty) Center(child: Text(w.headline)),
      if (w != null)
        Center(
          child: Text(
            '⭐ ${w.rating.toStringAsFixed(1)} · ${w.completedJobs} jobs réalisés · ${w.completionRate.toStringAsFixed(0)} % de complétion',
          ),
        ),
      if (u.zone.isNotEmpty) Center(child: Text('📍 ${u.zone}')),
      if (w != null)
        Center(
          child: Text(
            '${w.available ? '🟢 Disponible' : '⚪ Indisponible'}${w.hourlyRate > 0 ? ' · ${fcfa(w.hourlyRate)}/h' : ''}',
          ),
        ),
      const SizedBox(height: 8),
      Wrap(
        alignment: WrapAlignment.center,
        spacing: 6,
        children: [
          for (final b in w?.badges ?? (u.identityVerified ? ['Identité vérifiée'] : <String>[]))
            Chip(label: Text(b == 'Top Worker' ? '🏆 $b' : '✅ $b')),
        ],
      ),
      if (u.bio.isNotEmpty)
        Padding(
          padding: const EdgeInsets.all(8),
          child: Text(u.bio, textAlign: TextAlign.center),
        ),
      if (w != null && w.skills.isNotEmpty) ...[
        const Text('Compétences', style: TextStyle(fontWeight: FontWeight.bold)),
        Wrap(spacing: 6, children: [for (final s in w.skills) Chip(label: Text(s.verified ? '${s.name} ✓' : s.name))]),
      ],
      const SizedBox(height: 12),
      Text('Avis (${reviews.length})', style: const TextStyle(fontWeight: FontWeight.bold)),
      if (reviews.isEmpty) const Text("Pas encore d'avis."),
      for (final r in reviews)
        ListTile(
          contentPadding: EdgeInsets.zero,
          title: Text('${'⭐' * r.rating} ${r.reviewerName}'),
          subtitle: Text([r.jobTitle, if (r.comment.isNotEmpty) r.comment].join('\n')),
          trailing: Text(ago(r.createdAt), style: const TextStyle(fontSize: 11)),
        ),
    ];
  }
}
