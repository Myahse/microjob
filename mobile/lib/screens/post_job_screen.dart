import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:provider/provider.dart';

import '../api/models.dart';
import '../state/session.dart';
import '../util/format.dart';
import 'job_detail_screen.dart';

class PostJobScreen extends StatefulWidget {
  const PostJobScreen({super.key, this.categoryId});
  final int? categoryId;

  @override
  State<PostJobScreen> createState() => _PostJobScreenState();
}

class _PostJobScreenState extends State<PostJobScreen> {
  List<Category> categories = [];
  int? categoryId;
  int? skillId;
  final title = TextEditingController();
  final description = TextEditingController();
  final address = TextEditingController();
  final budget = TextEditingController();
  final duration = TextEditingController(text: '60');
  String zone = zones.first.name;
  double? gpsLat, gpsLng;
  DateTime? date;
  TimeOfDay? time;
  String mode = 'fixed';
  final photos = <String>[];
  bool busy = false;

  @override
  void initState() {
    super.initState();
    categoryId = widget.categoryId;
    final s = context.read<Session>();
    if (zones.any((z) => z.name == s.user!.zone)) zone = s.user!.zone;
    s.api.categories().then((c) => mounted ? setState(() => categories = c) : null);
  }

  Category? get category => categories.where((c) => c.id == categoryId).firstOrNull;

  Future<void> pickPhoto() async {
    final api = context.read<Session>().api;
    final img = await ImagePicker().pickImage(source: ImageSource.gallery, maxWidth: 1600, imageQuality: 80);
    if (img == null) return;
    try {
      final url = await api.upload(await img.readAsBytes(), img.name);
      setState(() => photos.add(url));
    } catch (e) {
      if (mounted) toast(context, e);
    }
  }

  Future<void> useGps() async {
    try {
      final p = await currentPosition();
      setState(() => (gpsLat = p.latitude, gpsLng = p.longitude));
    } catch (e) {
      if (mounted) toast(context, e);
    }
  }

  Future<void> submit() async {
    if (categoryId == null) return toast(context, 'Choisissez une catégorie');
    final api = context.read<Session>().api;
    final z = zoneNamed(zone);
    DateTime? at;
    if (date != null) {
      final t = time ?? const TimeOfDay(hour: 9, minute: 0);
      at = DateTime(date!.year, date!.month, date!.day, t.hour, t.minute);
    }
    setState(() => busy = true);
    try {
      final (job, notified) = await api.createJob({
        'category_id': categoryId,
        'skill_id': skillId,
        'title': title.text,
        'description': description.text,
        'latitude': gpsLat ?? z.lat,
        'longitude': gpsLng ?? z.lng,
        'address': address.text,
        'zone': zone,
        'pricing_mode': mode,
        'budget': int.tryParse(budget.text) ?? 0,
        'scheduled_at': at?.toUtc().toIso8601String(),
        'duration_minutes': int.tryParse(duration.text) ?? 60,
        'photos': photos,
      });
      if (!mounted) return;
      toast(context, '$notified worker(s) compatible(s) notifié(s)');
      Navigator.pushReplacement(context, MaterialPageRoute(builder: (_) => JobDetailScreen(jobId: job.id)));
    } catch (e) {
      if (mounted) toast(context, e);
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final api = context.read<Session>().api;
    const gap = SizedBox(height: 12);
    return Scaffold(
      appBar: AppBar(title: const Text('Poster un job')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          const Text('Catégorie', style: TextStyle(fontWeight: FontWeight.w600)),
          const SizedBox(height: 6),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              for (final c in categories)
                ChoiceChip(
                  label: Text('${c.icon} ${c.name}'),
                  selected: categoryId == c.id,
                  onSelected: (_) => setState(() => (categoryId = c.id, skillId = null)),
                ),
            ],
          ),
          gap,
          if (category != null) ...[
            DropdownButtonFormField<int?>(
              initialValue: skillId,
              decoration: const InputDecoration(labelText: 'Type de service'),
              items: [
                const DropdownMenuItem(value: null, child: Text('Autre / non précisé')),
                for (final s in category!.skills) DropdownMenuItem(value: s.id, child: Text(s.name)),
              ],
              onChanged: (v) => setState(() => skillId = v),
            ),
            gap,
          ],
          TextField(
            controller: title,
            decoration: const InputDecoration(labelText: 'Titre', hintText: 'Installer Windows sur mon PC'),
          ),
          gap,
          TextField(
            controller: description,
            maxLines: 3,
            decoration: const InputDecoration(labelText: 'Description'),
          ),
          gap,
          DropdownButtonFormField<String>(
            initialValue: zone,
            decoration: const InputDecoration(labelText: '📍 Quartier (visible par les workers)'),
            items: [for (final z in zones) DropdownMenuItem(value: z.name, child: Text(z.name))],
            onChanged: (v) => setState(() => zone = v!),
          ),
          gap,
          TextField(
            controller: address,
            decoration: const InputDecoration(
              labelText: 'Adresse exacte',
              helperText: 'Partagée uniquement avec le worker choisi',
            ),
          ),
          Align(
            alignment: Alignment.centerLeft,
            child: TextButton.icon(
              onPressed: useGps,
              icon: Icon(gpsLat == null ? Icons.my_location : Icons.check_circle),
              label: Text(gpsLat == null ? 'Utiliser ma position exacte' : 'Position GPS enregistrée'),
            ),
          ),
          Row(
            children: [
              Expanded(
                child: OutlinedButton.icon(
                  icon: const Icon(Icons.event),
                  label: Text(date == null ? "Aujourd'hui / dès que possible" : '${date!.day}/${date!.month}'),
                  onPressed: () async {
                    final d = await showDatePicker(
                      context: context,
                      firstDate: DateTime.now(),
                      lastDate: DateTime.now().add(const Duration(days: 60)),
                    );
                    if (d != null) setState(() => date = d);
                  },
                ),
              ),
              const SizedBox(width: 8),
              OutlinedButton.icon(
                icon: const Icon(Icons.schedule),
                label: Text(time == null ? 'Heure' : time!.format(context)),
                onPressed: () async {
                  final t = await showTimePicker(context: context, initialTime: const TimeOfDay(hour: 18, minute: 0));
                  if (t != null) setState(() => (time = t, date = date ?? DateTime.now()));
                },
              ),
            ],
          ),
          gap,
          TextField(
            controller: duration,
            keyboardType: TextInputType.number,
            decoration: const InputDecoration(labelText: 'Durée estimée (min)'),
          ),
          gap,
          SegmentedButton<String>(
            segments: const [
              ButtonSegment(value: 'fixed', label: Text('💰 Prix fixe')),
              ButtonSegment(value: 'offers', label: Text('🤝 Proposez votre prix')),
            ],
            selected: {mode},
            onSelectionChanged: (v) => setState(() => mode = v.first),
          ),
          gap,
          TextField(
            controller: budget,
            keyboardType: TextInputType.number,
            decoration: InputDecoration(labelText: mode == 'fixed' ? 'Budget (FCFA)' : 'Budget indicatif (optionnel)'),
          ),
          gap,
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              for (final p in photos)
                ClipRRect(
                  borderRadius: BorderRadius.circular(8),
                  child: Image.network(api.media(p), width: 64, height: 64, fit: BoxFit.cover),
                ),
              if (photos.length < 6)
                InkWell(
                  onTap: pickPhoto,
                  child: Container(
                    width: 64,
                    height: 64,
                    decoration: BoxDecoration(
                      border: Border.all(color: Colors.grey),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: const Icon(Icons.add_a_photo_outlined),
                  ),
                ),
            ],
          ),
          const SizedBox(height: 20),
          FilledButton(
            onPressed: busy ? null : submit,
            style: FilledButton.styleFrom(padding: const EdgeInsets.all(16)),
            child: const Text('POST JOB'),
          ),
        ],
      ),
    );
  }
}
