import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../api/models.dart';
import '../state/session.dart';
import '../widgets/common.dart';
import 'post_job_screen.dart';
import 'search_screen.dart';

class ClientHome extends StatefulWidget {
  const ClientHome({super.key});

  @override
  State<ClientHome> createState() => _ClientHomeState();
}

class _ClientHomeState extends State<ClientHome> {
  int? nearby;
  List<Category> categories = [];

  @override
  void initState() {
    super.initState();
    final s = context.read<Session>();
    s.api.categories().then((c) => mounted ? setState(() => categories = c) : null).catchError((_) => null);
    final u = s.user!;
    if (u.latitude != null) {
      s.api
          .nearbyCount(u.latitude!, u.longitude!)
          .then((n) => mounted ? setState(() => nearby = n) : null)
          .catchError((_) => null);
    }
  }

  @override
  Widget build(BuildContext context) {
    final api = context.read<Session>().api;
    final user = context.watch<Session>().user!;
    return AsyncList<Job>(
      load: () => api.myJobs('active'),
      reloadOn: (e) => e.type == 'job',
      empty: 'Aucun job en cours.\nBesoin de quelqu\'un ? Postez le job.',
      header: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Container(
            margin: const EdgeInsets.all(12),
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              gradient: const LinearGradient(colors: [Color(0xFF4F46E5), Color(0xFF7C3AED)]),
              borderRadius: BorderRadius.circular(16),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Bonjour ${user.firstName} 👋', style: const TextStyle(color: Colors.white70)),
                const SizedBox(height: 4),
                const Text(
                  "Besoin de quelqu'un ?\nPostez le job.",
                  style: TextStyle(color: Colors.white, fontSize: 22, fontWeight: FontWeight.w800),
                ),
                const SizedBox(height: 6),
                Text(
                  nearby == null ? 'Des workers près de chez vous' : '$nearby workers disponibles à moins de 5 km',
                  style: const TextStyle(color: Colors.white),
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    FilledButton.tonal(
                      onPressed: () => Navigator.push(context, MaterialPageRoute(builder: (_) => const PostJobScreen())),
                      child: const Text('POST A JOB'),
                    ),
                    const SizedBox(width: 8),
                    TextButton(
                      onPressed: () => Navigator.push(context, MaterialPageRoute(builder: (_) => const SearchScreen())),
                      style: TextButton.styleFrom(foregroundColor: Colors.white),
                      child: const Text('Trouver un prestataire'),
                    ),
                  ],
                ),
              ],
            ),
          ),
          SizedBox(
            height: 92,
            child: ListView(
              scrollDirection: Axis.horizontal,
              padding: const EdgeInsets.symmetric(horizontal: 8),
              children: [
                for (final c in categories)
                  Card(
                    child: InkWell(
                      onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => PostJobScreen(categoryId: c.id))),
                      child: SizedBox(
                        width: 92,
                        child: Column(
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: [
                            Text(c.icon, style: const TextStyle(fontSize: 28)),
                            Text(c.name, style: const TextStyle(fontWeight: FontWeight.w600)),
                          ],
                        ),
                      ),
                    ),
                  ),
              ],
            ),
          ),
          const Padding(
            padding: EdgeInsets.fromLTRB(16, 16, 16, 4),
            child: Text('Mes jobs en cours', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
          ),
        ],
      ),
      itemBuilder: (context, job, reload) => JobTile(job, onReturn: reload),
    );
  }
}
