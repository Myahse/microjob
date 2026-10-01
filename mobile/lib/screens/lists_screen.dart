import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../api/models.dart';
import '../state/session.dart';
import '../widgets/common.dart';

class _TabbedJobs extends StatelessWidget {
  const _TabbedJobs({required this.tabs, required this.load, this.reloadOn});
  final List<(String, String)> tabs;
  final Future<List<Job>> Function(String scope) load;
  final bool Function(RealtimeEvent)? reloadOn;

  @override
  Widget build(BuildContext context) {
    return DefaultTabController(
      length: tabs.length,
      child: Column(
        children: [
          TabBar(
            isScrollable: tabs.length > 3,
            tabs: [for (final t in tabs) Tab(text: t.$2)],
          ),
          Expanded(
            child: TabBarView(
              children: [
                for (final t in tabs)
                  AsyncList<Job>(
                    key: PageStorageKey(t.$1),
                    load: () => load(t.$1),
                    reloadOn: reloadOn,
                    itemBuilder: (context, job, reload) => JobTile(job, onReturn: reload),
                  ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

/// Client: "Mes jobs" — actifs, terminés, annulés.
class MyJobsScreen extends StatelessWidget {
  const MyJobsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final api = context.read<Session>().api;
    return _TabbedJobs(
      tabs: const [('active', 'Actifs'), ('done', 'Terminés'), ('cancelled', 'Annulés')],
      load: api.myJobs,
      reloadOn: (e) => e.type == 'job',
    );
  }
}

/// Worker: "Mes missions" — disponibles, offres, en cours, terminées.
class MissionsScreen extends StatelessWidget {
  const MissionsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final api = context.read<Session>().api;
    return _TabbedJobs(
      tabs: const [('available', 'Disponibles'), ('applied', 'Offres'), ('active', 'En cours'), ('done', 'Terminées')],
      load: (scope) => scope == 'available' ? api.availableJobs() : api.missions(scope),
      reloadOn: (e) => e.type == 'job',
    );
  }
}
