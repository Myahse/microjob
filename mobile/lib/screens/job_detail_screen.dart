import 'dart:async';

import 'package:flutter/material.dart';
import 'package:geolocator/geolocator.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';

import '../api/models.dart';
import '../state/session.dart';
import '../util/format.dart';
import '../widgets/common.dart';
import 'chat_screen.dart';
import 'public_profile_screen.dart';

const _active = ['ASSIGNED', 'WORKER_ON_THE_WAY', 'ARRIVED', 'STARTED', 'AWAITING_CONFIRMATION'];
const _finished = ['COMPLETED', 'PAID', 'RATED'];
const _cancellable = ['ASSIGNED', 'WORKER_ON_THE_WAY', 'ARRIVED'];
const _timeline = ['MATCHING', 'ASSIGNED', 'WORKER_ON_THE_WAY', 'ARRIVED', 'STARTED', 'AWAITING_CONFIRMATION', 'PAID', 'RATED'];

class JobDetailScreen extends StatefulWidget {
  const JobDetailScreen({super.key, required this.jobId});
  final String jobId;

  @override
  State<JobDetailScreen> createState() => _JobDetailScreenState();
}

class _JobDetailScreenState extends State<JobDetailScreen> {
  Job? job;
  String? error;
  List<Application> applications = [];
  StreamSubscription? sub;
  StreamSubscription<Position>? locationSub;
  DateTime lastShare = DateTime(2000);

  @override
  void initState() {
    super.initState();
    load();
    sub = context.read<Session>().events.listen((e) {
      if (e.type == 'job' && e.data['job_id'] == widget.jobId) load();
    });
  }

  @override
  void dispose() {
    sub?.cancel();
    locationSub?.cancel();
    super.dispose();
  }

  Future<void> load() async {
    final api = context.read<Session>().api;
    try {
      final j = await api.job(widget.jobId);
      List<Application> apps = [];
      if (j.myRole == 'client' && j.isOpen && j.pricingMode == 'offers') apps = await api.applications(j.id);
      if (mounted) setState(() => (job = j, applications = apps, error = null));
      if (!_cancellable.contains(j.status)) await locationSub?.cancel();
    } catch (e) {
      if (mounted) setState(() => error = e.toString());
    }
  }

  Future<void> run(Future<Object?> Function() action, {String? success}) async {
    try {
      final res = await action();
      if (res is Job && mounted) setState(() => job = res);
      await load();
      if (success != null && mounted) toast(context, success);
    } catch (e) {
      if (mounted) toast(context, e);
    }
  }

  Future<String?> ask(String title, {String hint = '', TextInputType? type}) {
    final c = TextEditingController();
    return showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(title),
        content: TextField(
          controller: c,
          keyboardType: type,
          autofocus: true,
          decoration: InputDecoration(hintText: hint),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Annuler')),
          FilledButton(onPressed: () => Navigator.pop(ctx, c.text.trim()), child: const Text('OK')),
        ],
      ),
    );
  }

  /// Shares the worker's live position with the client while travelling.
  void toggleSharing(bool on) {
    final api = context.read<Session>().api;
    if (!on) {
      locationSub?.cancel();
      setState(() => locationSub = null);
      return;
    }
    currentPosition()
        .then((_) {
          setState(() {
            locationSub = Geolocator.getPositionStream(locationSettings: const LocationSettings(distanceFilter: 50)).listen((p) {
              if (DateTime.now().difference(lastShare).inSeconds < 15) return;
              lastShare = DateTime.now();
              api.shareLocation(widget.jobId, p.latitude, p.longitude).catchError((_) {});
            });
          });
        })
        .catchError((Object e) {
          if (mounted) toast(context, e);
        });
  }

  @override
  Widget build(BuildContext context) {
    final j = job;
    return Scaffold(
      appBar: AppBar(title: Text(j?.title ?? 'Job')),
      body: j == null
          ? Center(child: error != null ? Text(error!) : const CircularProgressIndicator())
          : RefreshIndicator(
              onRefresh: load,
              child: ListView(padding: const EdgeInsets.all(12), children: _content(j)),
            ),
      bottomNavigationBar: j == null ? null : _actions(j),
    );
  }

  List<Widget> _content(Job j) {
    final api = context.read<Session>().api;
    final price = j.pricingMode == 'offers' && j.agreedPrice == null ? 'Sur offres' : fcfa(j.price);
    return [
      Card(
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Expanded(child: Text('${j.categoryIcon} ${j.categoryName}${j.skillName != null ? ' · ${j.skillName}' : ''}')),
                  StatusChip(j.status),
                ],
              ),
              const SizedBox(height: 6),
              Text(j.title, style: Theme.of(context).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.bold)),
              const SizedBox(height: 8),
              Wrap(
                spacing: 14,
                runSpacing: 6,
                children: [
                  Text('📍 ${j.zone}'),
                  Text('💰 $price'),
                  Text('🕕 ${when(j.scheduledAt)}'),
                  Text('⏳ ${j.durationMinutes} min'),
                  if (j.distanceKm != null) Text('📏 ${km(j.distanceKm)}'),
                ],
              ),
              if (j.address != null && j.address!.isNotEmpty) ...[const SizedBox(height: 6), Text('🏠 ${j.address}')],
              if (j.description.isNotEmpty) ...[const SizedBox(height: 10), Text(j.description)],
              if (j.photos.isNotEmpty) ...[
                const SizedBox(height: 10),
                SizedBox(
                  height: 90,
                  child: ListView(
                    scrollDirection: Axis.horizontal,
                    children: [
                      for (final p in j.photos)
                        Padding(
                          padding: const EdgeInsets.only(right: 8),
                          child: ClipRRect(
                            borderRadius: BorderRadius.circular(8),
                            child: Image.network(api.media(p), width: 90, fit: BoxFit.cover),
                          ),
                        ),
                    ],
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
      if (j.myRole != 'candidate' && j.status != 'CANCELLED' && j.status != 'DISPUTED') _timelineCard(j),
      if (j.myRole == 'client' && j.isOpen)
        Card(
          color: Colors.amber.shade50,
          child: ListTile(
            leading: const Icon(Icons.radar),
            title: const Text('Recherche de workers à proximité…'),
            subtitle: Text(
              j.pricingMode == 'fixed' ? 'Le premier qui accepte est attribué.' : 'Choisissez parmi les offres reçues.',
            ),
          ),
        ),
      if (j.myRole == 'client' && j.isOpen && j.pricingMode == 'offers') ..._applications(j),
      if (j.worker != null && j.myRole != 'worker')
        _personCard(j.worker!, 'Worker', distance: j.assignment?.workerDistanceKm, j: j),
      if (j.myRole == 'worker') _personCard(j.client, 'Client', j: j),
      if (j.payment != null && j.myRole != 'candidate') _paymentCard(j.payment!),
      if (j.openReport)
        const Card(
          child: ListTile(leading: Icon(Icons.gavel), title: Text("Un signalement est en cours d'examen.")),
        ),
      if (j.myRole == 'candidate' && j.myApplication?.status == 'pending')
        Card(
          child: ListTile(
            title: Text('Offre envoyée : ${fcfa(j.myApplication!.price)}'),
            subtitle: const Text('Vous pouvez la modifier.'),
          ),
        ),
      if (j.history.isNotEmpty)
        Card(
          child: ExpansionTile(
            title: const Text('Historique'),
            children: [
              for (final h in j.history)
                ListTile(
                  dense: true,
                  title: Text(statusLabel[h.status] ?? h.status),
                  subtitle: h.note.isEmpty ? null : Text(h.note),
                  trailing: Text(ago(h.createdAt), style: const TextStyle(fontSize: 11)),
                ),
            ],
          ),
        ),
      const SizedBox(height: 24),
    ];
  }

  Widget _timelineCard(Job j) {
    final current = j.status == 'POSTED' ? 'MATCHING' : (j.status == 'COMPLETED' ? 'PAID' : j.status);
    final idx = _timeline.indexOf(current);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Wrap(
          spacing: 6,
          runSpacing: 6,
          children: [
            for (var i = 0; i < _timeline.length; i++)
              Chip(
                visualDensity: VisualDensity.compact,
                label: Text(statusLabel[_timeline[i]]!, style: TextStyle(fontSize: 11, color: i == idx ? Colors.white : null)),
                backgroundColor: i < idx ? Colors.green.shade100 : (i == idx ? Colors.indigo : Colors.grey.shade100),
              ),
          ],
        ),
      ),
    );
  }

  List<Widget> _applications(Job j) => [
    Padding(
      padding: const EdgeInsets.fromLTRB(4, 12, 4, 4),
      child: Text(
        'Workers intéressés (${applications.where((a) => a.status == 'pending').length})',
        style: const TextStyle(fontWeight: FontWeight.bold),
      ),
    ),
    for (final a in applications.where((a) => a.status == 'pending'))
      Card(
        child: ListTile(
          leading: UserAvatar(url: a.worker!.avatarUrl, name: a.worker!.firstName),
          title: Text('${a.worker!.fullName} · ${fcfa(a.price)}'),
          subtitle: Text(
            [
              '⭐ ${a.worker!.rating.toStringAsFixed(1)} (${a.worker!.ratingCount})',
              '${a.worker!.completedJobs} jobs',
              if (a.distanceKm != null) km(a.distanceKm),
              if (a.message.isNotEmpty) '« ${a.message} »',
            ].join(' · '),
          ),
          trailing: FilledButton(
            onPressed: () =>
                run(() => context.read<Session>().api.choose(j.id, a.id), success: '${a.worker!.firstName} est attribué'),
            child: const Text('Choisir'),
          ),
        ),
      ),
  ];

  Widget _personCard(PersonRef p, String role, {double? distance, required Job j}) => Card(
    child: ListTile(
      leading: UserAvatar(url: p.avatarUrl, name: p.firstName),
      title: Text('$role : ${p.fullName}'),
      subtitle: Stars(p.rating, p.ratingCount),
      trailing: distance != null && _cancellable.contains(j.status)
          ? Text(
              '${p.firstName} est à\n${km(distance)}',
              textAlign: TextAlign.right,
              style: const TextStyle(color: Colors.indigo),
            )
          : null,
      onTap: role == 'Worker'
          ? () => Navigator.push(context, MaterialPageRoute(builder: (_) => PublicProfileScreen(userId: p.id)))
          : null,
    ),
  );

  Widget _paymentCard(Payment p) {
    const labels = {
      'held': '🔒 Retenu (séquestre)',
      'released': '✅ Versé',
      'refunded': '↩️ Remboursé',
      'partially_refunded': '↩️ Partiellement remboursé',
    };
    Widget row(String a, String b) => Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Row(
        children: [
          Expanded(
            child: Text(a, style: const TextStyle(color: Colors.grey)),
          ),
          Text(b),
        ],
      ),
    );
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('Paiement', style: TextStyle(fontWeight: FontWeight.bold)),
            row('Montant', fcfa(p.amount)),
            row('Commission (${(p.commissionRate * 100).round()} %)', fcfa(p.platformFee)),
            row('Worker reçoit', fcfa(p.workerAmount)),
            if (p.refundedAmount > 0) row('Remboursé', fcfa(p.refundedAmount)),
            row('Statut', labels[p.status] ?? p.status),
          ],
        ),
      ),
    );
  }

  /// The sticky action bar: what the viewer can do right now.
  Widget? _actions(Job j) {
    final api = context.read<Session>().api;
    final buttons = <Widget>[];
    void primary(String label, Future<Object?> Function() fn, {String? success}) => buttons.add(
      Expanded(
        child: FilledButton(
          onPressed: () => run(fn, success: success),
          child: Text(label),
        ),
      ),
    );
    void secondary(String label, VoidCallback fn) => buttons.add(
      Expanded(
        child: OutlinedButton(onPressed: fn, child: Text(label)),
      ),
    );

    final chatOpen = j.hasConversation;
    if (chatOpen) {
      buttons.add(
        IconButton.filledTonal(
          onPressed: () => Navigator.push(
            context,
            MaterialPageRoute(
              builder: (_) => ChatScreen(jobId: j.id, title: j.title, closed: j.status == 'CANCELLED'),
            ),
          ),
          icon: const Icon(Icons.chat_bubble_outline),
        ),
      );
    }

    switch (j.myRole) {
      case 'candidate':
        if (j.isOpen && j.pricingMode == 'fixed') {
          secondary(
            'DECLINE',
            () => run(() async {
              await api.decline(j.id);
              if (mounted) Navigator.pop(context);
              return null;
            }),
          );
          primary('ACCEPT JOB', () => api.accept(j.id), success: 'Job accepté ✅');
        } else if (j.isOpen) {
          secondary(
            'Pas intéressé',
            () => run(() async {
              await api.decline(j.id);
              if (mounted) Navigator.pop(context);
              return null;
            }),
          );
          primary('Proposer un prix', () async {
            final v = await ask('Votre prix (FCFA)', type: TextInputType.number);
            final price = int.tryParse(v ?? '');
            if (price == null) return null;
            return api.apply(j.id, price, '');
          }, success: 'Offre envoyée');
        }
      case 'worker':
        switch (j.status) {
          case 'ASSIGNED':
            secondary('Commencer (à distance)', () => run(() => api.advance(j.id, 'STARTED')));
            primary('🚗 En route', () => api.advance(j.id, 'WORKER_ON_THE_WAY'));
          case 'WORKER_ON_THE_WAY':
            buttons.add(
              IconButton(
                tooltip: 'Partager ma position',
                onPressed: () => toggleSharing(locationSub == null),
                icon: Icon(
                  locationSub == null ? Icons.location_off_outlined : Icons.location_on,
                  color: locationSub == null ? null : Colors.green,
                ),
              ),
            );
            if (j.latitude != null) {
              buttons.add(
                IconButton(
                  tooltip: 'Itinéraire',
                  onPressed: () =>
                      launchUrl(Uri.parse('https://www.google.com/maps/dir/?api=1&destination=${j.latitude},${j.longitude}')),
                  icon: const Icon(Icons.directions),
                ),
              );
            }
            primary('📍 ARRIVED', () => api.advance(j.id, 'ARRIVED'));
          case 'ARRIVED':
            primary('🛠 START JOB', () => api.advance(j.id, 'STARTED'));
          case 'STARTED':
            primary('🏁 COMPLETE JOB', () => api.advance(j.id, 'AWAITING_CONFIRMATION'));
        }
      case 'client':
        if (j.status == 'AWAITING_CONFIRMATION') {
          primary('CONFIRM COMPLETION', () => api.confirm(j.id), success: 'Paiement libéré au worker');
        }
    }

    final mine = j.myRole == 'client' || j.myRole == 'worker';
    if (mine && _finished.contains(j.status) && !j.reviewedByMe) {
      primary('⭐ Évaluer', () async {
        final rating = await _askRating();
        if (rating == null) return null;
        return api.review(j.id, {'rating': rating.$1, 'comment': rating.$2});
      }, success: 'Merci pour votre évaluation');
    }

    final canCancel =
        (j.myRole == 'client' && (j.isOpen || _cancellable.contains(j.status))) ||
        (j.myRole == 'worker' && _cancellable.contains(j.status));
    final canReport = mine && (_active.contains(j.status) || _finished.contains(j.status)) && !j.openReport;
    if (canCancel || canReport) {
      buttons.insert(
        0,
        PopupMenuButton<String>(
          onSelected: (v) async {
            if (v == 'cancel') {
              final reason = await ask("Raison de l'annulation");
              if (reason != null && reason.isNotEmpty) run(() => api.cancel(j.id, reason));
            } else {
              final r = await _askReport();
              if (r != null) run(() => api.report(j.id, r.$1, r.$2), success: 'Signalement envoyé');
            }
          },
          itemBuilder: (_) => [
            if (canCancel) const PopupMenuItem(value: 'cancel', child: Text('Annuler le job')),
            if (canReport) const PopupMenuItem(value: 'report', child: Text('⚠️ Signaler un problème')),
          ],
        ),
      );
    }

    if (buttons.isEmpty) return null;
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Row(
          children: [
            for (final b in buttons) ...[b, const SizedBox(width: 8)],
          ]..removeLast(),
        ),
      ),
    );
  }

  Future<(int, String)?> _askRating() {
    var rating = 5;
    final comment = TextEditingController();
    return showDialog<(int, String)>(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, set) => AlertDialog(
          title: const Text('Votre évaluation'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  for (var i = 1; i <= 5; i++)
                    IconButton(
                      onPressed: () => set(() => rating = i),
                      icon: Icon(i <= rating ? Icons.star : Icons.star_border, color: Colors.amber, size: 32),
                    ),
                ],
              ),
              TextField(
                controller: comment,
                decoration: const InputDecoration(hintText: 'Un commentaire ?'),
              ),
            ],
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Plus tard')),
            FilledButton(onPressed: () => Navigator.pop(ctx, (rating, comment.text)), child: const Text('Envoyer')),
          ],
        ),
      ),
    );
  }

  Future<(String, String)?> _askReport() {
    var type = 'not_done';
    final text = TextEditingController();
    return showDialog<(String, String)>(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, set) => AlertDialog(
          title: const Text('Report a problem'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              DropdownButtonFormField<String>(
                initialValue: type,
                items: [for (final e in reportTypes.entries) DropdownMenuItem(value: e.key, child: Text(e.value))],
                onChanged: (v) => set(() => type = v!),
              ),
              const SizedBox(height: 8),
              TextField(
                controller: text,
                maxLines: 3,
                decoration: const InputDecoration(hintText: 'Décrivez le problème'),
              ),
            ],
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Annuler')),
            FilledButton(onPressed: () => Navigator.pop(ctx, (type, text.text)), child: const Text('Envoyer')),
          ],
        ),
      ),
    );
  }
}
