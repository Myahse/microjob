import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../api/models.dart';
import '../screens/job_detail_screen.dart';
import '../state/session.dart';
import '../util/format.dart';

class StatusChip extends StatelessWidget {
  const StatusChip(this.status, {super.key});
  final String status;

  @override
  Widget build(BuildContext context) {
    final c = statusColor(status);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(color: c.withValues(alpha: 0.12), borderRadius: BorderRadius.circular(20)),
      child: Text(
        statusLabel[status] ?? status,
        style: TextStyle(color: c, fontSize: 12, fontWeight: FontWeight.w600),
      ),
    );
  }
}

class Stars extends StatelessWidget {
  const Stars(this.rating, this.count, {super.key});
  final double rating;
  final int count;

  @override
  Widget build(BuildContext context) {
    if (count == 0) return const Text('Nouveau', style: TextStyle(fontSize: 12, color: Colors.grey));
    return Text('⭐ ${rating.toStringAsFixed(1)} ($count)', style: TextStyle(fontSize: 13, color: Colors.amber.shade800));
  }
}

class UserAvatar extends StatelessWidget {
  const UserAvatar({super.key, required this.url, required this.name, this.radius = 20});
  final String url;
  final String name;
  final double radius;

  @override
  Widget build(BuildContext context) {
    final api = context.read<Session>().api;
    return CircleAvatar(
      radius: radius,
      backgroundImage: url.isEmpty ? null : NetworkImage(api.media(url)),
      child: url.isEmpty ? Text(name.isEmpty ? '?' : name[0].toUpperCase()) : null,
    );
  }
}

class JobTile extends StatelessWidget {
  const JobTile(this.job, {super.key, this.onReturn});
  final Job job;
  final VoidCallback? onReturn;

  @override
  Widget build(BuildContext context) {
    final price = job.pricingMode == 'offers' && job.agreedPrice == null ? 'Proposez votre prix' : fcfa(job.price);
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 5),
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: () async {
          await Navigator.push(context, MaterialPageRoute(builder: (_) => JobDetailScreen(jobId: job.id)));
          onReturn?.call();
        },
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Text(
                      '${job.categoryIcon} ${job.categoryName}${job.skillName != null ? ' · ${job.skillName}' : ''}',
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                  ),
                  StatusChip(job.status),
                ],
              ),
              const SizedBox(height: 4),
              Text(job.title, style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
              const SizedBox(height: 6),
              Wrap(
                spacing: 12,
                runSpacing: 4,
                children: [
                  Text('📍 ${job.zone}'),
                  Text('💰 $price'),
                  if (job.distanceKm != null) Text('📏 ${km(job.distanceKm)}'),
                  Text('⏱ ${when(job.scheduledAt)}'),
                  if (job.myRole == 'client' && job.pricingMode == 'offers' && job.applicationsCount > 0)
                    Text(
                      '${job.applicationsCount} offre(s)',
                      style: const TextStyle(color: Colors.indigo, fontWeight: FontWeight.w600),
                    ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class EmptyState extends StatelessWidget {
  const EmptyState(this.text, {super.key, this.icon = Icons.inbox_outlined});
  final String text;
  final IconData icon;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.all(40),
    child: Column(
      children: [
        Icon(icon, size: 48, color: Colors.grey.shade400),
        const SizedBox(height: 8),
        Text(
          text,
          textAlign: TextAlign.center,
          style: TextStyle(color: Colors.grey.shade600),
        ),
      ],
    ),
  );
}

/// Loads a list and shows loading / error / empty states with pull-to-refresh.
/// Give it a new [Key] to reload with different parameters.
class AsyncList<T> extends StatefulWidget {
  const AsyncList({
    super.key,
    required this.load,
    required this.itemBuilder,
    this.empty = 'Rien ici pour le moment.',
    this.header,
    this.reloadOn,
  });
  final Future<List<T>> Function() load;
  final Widget Function(BuildContext, T, VoidCallback reload) itemBuilder;
  final String empty;
  final Widget? header;
  final bool Function(RealtimeEvent)? reloadOn;

  @override
  State<AsyncList<T>> createState() => _AsyncListState<T>();
}

class _AsyncListState<T> extends State<AsyncList<T>> {
  List<T>? items;
  String? error;
  late final Session session;
  dynamic sub;

  @override
  void initState() {
    super.initState();
    session = context.read<Session>();
    if (widget.reloadOn != null) {
      sub = session.events.where(widget.reloadOn!).listen((_) => reload());
    }
    reload();
  }

  @override
  void dispose() {
    sub?.cancel();
    super.dispose();
  }

  Future<void> reload() async {
    try {
      final r = await widget.load();
      if (mounted) setState(() => (items = r, error = null));
    } catch (e) {
      if (mounted) setState(() => error = e.toString());
    }
  }

  @override
  Widget build(BuildContext context) {
    final children = <Widget>[
      if (widget.header != null) widget.header!,
      if (error != null)
        Padding(
          padding: const EdgeInsets.all(16),
          child: Text(error!, style: const TextStyle(color: Colors.red)),
        ),
      if (items == null && error == null)
        const Padding(
          padding: EdgeInsets.all(32),
          child: Center(child: CircularProgressIndicator()),
        ),
      if (items != null && items!.isEmpty) EmptyState(widget.empty),
      ...?items?.map((e) => widget.itemBuilder(context, e, reload)),
      const SizedBox(height: 80),
    ];
    return RefreshIndicator(
      onRefresh: reload,
      child: ListView(children: children),
    );
  }
}
