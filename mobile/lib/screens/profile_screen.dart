import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:provider/provider.dart';

import '../api/models.dart';
import '../state/session.dart';
import '../util/format.dart';
import '../widgets/common.dart';
import 'public_profile_screen.dart';
import 'search_screen.dart';
import 'wallet_screen.dart';

class ProfileScreen extends StatelessWidget {
  const ProfileScreen({super.key});

  Future<void> _avatar(BuildContext context) async {
    final s = context.read<Session>();
    final img = await ImagePicker().pickImage(source: ImageSource.gallery, maxWidth: 800, imageQuality: 85);
    if (img == null) return;
    try {
      final url = await s.api.upload(await img.readAsBytes(), img.name);
      s.setUser(await s.api.updateMe({'avatar_url': url}));
    } catch (e) {
      if (context.mounted) toast(context, e);
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = context.watch<Session>();
    final u = s.user!;
    final w = u.worker;
    return ListView(
      padding: const EdgeInsets.all(12),
      children: [
        Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Row(
              children: [
                GestureDetector(
                  onTap: () => _avatar(context),
                  child: UserAvatar(url: u.avatarUrl, name: u.firstName, radius: 34),
                ),
                const SizedBox(width: 14),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(u.fullName, style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
                      Text('@${u.username} · ${u.zone}'),
                      if (w != null) ...[
                        if (w.headline.isNotEmpty) Text(w.headline),
                        Text(
                          '⭐ ${w.rating.toStringAsFixed(1)} · ${w.completedJobs} jobs · ${w.completionRate.toStringAsFixed(0)} % complétion',
                        ),
                      ],
                      Wrap(
                        spacing: 4,
                        children: [
                          for (final b in w?.badges ?? <String>[]) Chip(label: Text(b), visualDensity: VisualDensity.compact),
                        ],
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
        if (u.isWorker)
          ListTile(
            leading: const Icon(Icons.account_balance_wallet_outlined),
            title: const Text('Revenus & retraits'),
            onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => const WalletScreen())),
          ),
        ListTile(
          leading: const Icon(Icons.handyman_outlined),
          title: Text(u.isWorker ? 'Mon profil worker & compétences' : 'Je veux proposer mes services'),
          onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => const WorkerSetupScreen())),
        ),
        ListTile(
          leading: const Icon(Icons.search),
          title: const Text('Trouver un prestataire'),
          onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => const SearchScreen())),
        ),
        ListTile(
          leading: const Icon(Icons.badge_outlined),
          title: const Text('Voir mon profil public'),
          onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => PublicProfileScreen(userId: u.id))),
        ),
        ListTile(leading: const Icon(Icons.logout), title: const Text('Déconnexion'), onTap: s.signOut),
      ],
    );
  }
}

/// Create or edit the professional profile: headline, rate, radius, skills.
class WorkerSetupScreen extends StatefulWidget {
  const WorkerSetupScreen({super.key});

  @override
  State<WorkerSetupScreen> createState() => _WorkerSetupScreenState();
}

class _WorkerSetupScreenState extends State<WorkerSetupScreen> {
  List<Category> categories = [];
  final headline = TextEditingController();
  final rate = TextEditingController(text: '0');
  double radius = 10;
  final skills = <int>{};

  @override
  void initState() {
    super.initState();
    final s = context.read<Session>();
    final w = s.user!.worker;
    if (w != null) {
      headline.text = w.headline;
      rate.text = '${w.hourlyRate}';
      radius = w.radiusKm.clamp(1, 50);
      skills.addAll(w.skills.map((e) => e.id));
    }
    s.api.categories().then((c) => mounted ? setState(() => categories = c) : null);
  }

  Future<void> save() async {
    final s = context.read<Session>();
    try {
      s.setUser(
        await s.api.updateWorker({
          'headline': headline.text,
          'hourly_rate': int.tryParse(rate.text) ?? 0,
          'radius_km': radius,
          'skill_ids': skills.toList(),
        }),
      );
      if (mounted) {
        toast(context, 'Profil worker enregistré');
        Navigator.pop(context);
      }
    } catch (e) {
      if (mounted) toast(context, e);
    }
  }

  @override
  Widget build(BuildContext context) {
    final verified = {
      for (final sk in context.read<Session>().user!.worker?.skills ?? <Skill>[])
        if (sk.verified) sk.id,
    };
    return Scaffold(
      appBar: AppBar(title: const Text('Profil worker')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          TextField(
            controller: headline,
            decoration: const InputDecoration(labelText: 'Titre', hintText: '💻 Développeur / Technicien'),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: rate,
            keyboardType: TextInputType.number,
            decoration: const InputDecoration(labelText: 'Tarif horaire (FCFA)'),
          ),
          const SizedBox(height: 12),
          Text("Rayon d'intervention : ${radius.round()} km"),
          Slider(value: radius, min: 1, max: 50, divisions: 49, onChanged: (v) => setState(() => radius = v)),
          const Text('Compétences', style: TextStyle(fontWeight: FontWeight.bold)),
          for (final c in categories) ...[
            Padding(padding: const EdgeInsets.only(top: 10, bottom: 4), child: Text('${c.icon} ${c.name}')),
            Wrap(
              spacing: 6,
              runSpacing: 6,
              children: [
                for (final sk in c.skills)
                  FilterChip(
                    label: Text(verified.contains(sk.id) ? '${sk.name} ✓' : sk.name),
                    selected: skills.contains(sk.id),
                    onSelected: (on) => setState(() => on ? skills.add(sk.id) : skills.remove(sk.id)),
                  ),
              ],
            ),
          ],
          const SizedBox(height: 20),
          FilledButton(onPressed: save, child: const Text('Enregistrer')),
        ],
      ),
    );
  }
}
