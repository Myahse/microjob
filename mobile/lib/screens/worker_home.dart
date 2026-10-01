import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../api/models.dart';
import '../state/session.dart';
import '../util/format.dart';
import '../widgets/common.dart';

/// Worker home: availability switch and nearby jobs feed.
class WorkerHome extends StatefulWidget {
  const WorkerHome({super.key});

  @override
  State<WorkerHome> createState() => _WorkerHomeState();
}

class _WorkerHomeState extends State<WorkerHome> {
  int feedKey = 0;

  Future<void> setAvailable(bool available, {bool gps = false, Zone? zone}) async {
    final s = context.read<Session>();
    try {
      double? lat, lng;
      String? zoneName;
      if (gps) {
        final p = await currentPosition();
        lat = p.latitude;
        lng = p.longitude;
      } else if (zone != null) {
        lat = zone.lat;
        lng = zone.lng;
        zoneName = zone.name;
      }
      s.setUser(await s.api.setAvailability(available, lat: lat, lng: lng, zone: zoneName));
      setState(() => feedKey++);
    } catch (e) {
      if (mounted) toast(context, e);
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = context.watch<Session>();
    final w = s.user!.worker!;
    return AsyncList<Job>(
      key: ValueKey(feedKey),
      load: s.api.availableJobs,
      reloadOn: (e) => e.type == 'job' || (e.type == 'notification' && e.data['kind'] == 'job_nearby'),
      empty: 'Aucun job à proximité pour le moment.',
      header: Card(
        margin: const EdgeInsets.all(12),
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              SwitchListTile(
                contentPadding: EdgeInsets.zero,
                value: w.available,
                onChanged: (v) => setAvailable(v, zone: s.user!.latitude == null ? zoneNamed(s.user!.zone) : null),
                title: Text(w.available ? '🟢 Disponible' : '⚪ Hors ligne', style: const TextStyle(fontWeight: FontWeight.bold)),
                subtitle: Text(
                  'Zone : ${s.user!.zone.isEmpty ? 'non définie' : s.user!.zone} · rayon ${w.radiusKm.toStringAsFixed(0)} km',
                ),
              ),
              Wrap(
                spacing: 8,
                children: [
                  OutlinedButton.icon(
                    onPressed: () => setAvailable(w.available, gps: true),
                    icon: const Icon(Icons.my_location, size: 18),
                    label: const Text('Ma position'),
                  ),
                  PopupMenuButton<Zone>(
                    onSelected: (z) => setAvailable(w.available, zone: z),
                    itemBuilder: (_) => [for (final z in zones) PopupMenuItem(value: z, child: Text(z.name))],
                    child: const Chip(avatar: Icon(Icons.place, size: 18), label: Text('Changer de quartier')),
                  ),
                ],
              ),
              if (w.skills.isEmpty)
                const Padding(
                  padding: EdgeInsets.only(top: 8),
                  child: Text(
                    'Ajoutez vos compétences dans Profil pour recevoir des jobs.',
                    style: TextStyle(color: Colors.orange),
                  ),
                ),
            ],
          ),
        ),
      ),
      itemBuilder: (context, job, reload) => JobTile(job, onReturn: reload),
    );
  }
}
