import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../state/session.dart';
import '../util/format.dart';

class AuthScreen extends StatefulWidget {
  const AuthScreen({super.key});

  @override
  State<AuthScreen> createState() => _AuthScreenState();
}

class _AuthScreenState extends State<AuthScreen> {
  bool register = false;
  bool busy = false;
  bool wants = true;
  bool offers = false;
  String zone = zones.first.name;
  final c = {
    for (final k in ['identifier', 'password', 'first_name', 'last_name', 'username', 'phone', 'email'])
      k: TextEditingController(),
  };

  Future<void> submit() async {
    final session = context.read<Session>();
    setState(() => busy = true);
    try {
      if (register) {
        if (!wants && !offers) throw Exception('Choisissez au moins un rôle');
        final z = zoneNamed(zone);
        final (token, user) = await session.api.register({
          'first_name': c['first_name']!.text,
          'last_name': c['last_name']!.text,
          'username': c['username']!.text,
          'phone': c['phone']!.text,
          'email': c['email']!.text,
          'password': c['password']!.text,
          'zone': z.name,
          'latitude': z.lat,
          'longitude': z.lng,
          'wants_services': wants,
          'offers_services': offers,
        });
        await session.signIn(token, user);
      } else {
        final (token, user) = await session.api.login(c['identifier']!.text.trim(), c['password']!.text);
        await session.signIn(token, user);
      }
    } catch (e) {
      if (mounted) toast(context, e);
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  Widget field(String key, String label, {bool obscure = false, TextInputType? type}) => Padding(
    padding: const EdgeInsets.only(bottom: 12),
    child: TextField(
      controller: c[key],
      obscureText: obscure,
      keyboardType: type,
      decoration: InputDecoration(labelText: label),
    ),
  );

  Widget roleCard(bool active, String icon, String label, VoidCallback onTap) => Expanded(
    child: InkWell(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          border: Border.all(color: active ? Colors.indigo : Colors.grey.shade300, width: 2),
          borderRadius: BorderRadius.circular(12),
          color: active ? Colors.indigo.shade50 : null,
        ),
        child: Column(
          children: [
            Text(icon, style: const TextStyle(fontSize: 24)),
            Text(label, textAlign: TextAlign.center),
          ],
        ),
      ),
    ),
  );

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.all(24),
          children: [
            const SizedBox(height: 24),
            Text(
              'MicroJob',
              style: TextStyle(fontSize: 34, fontWeight: FontWeight.w900, color: Colors.indigo.shade600),
            ),
            const Text("J'ai besoin de quelqu'un. Je poste. Quelqu'un proche accepte. Le job est fait."),
            const SizedBox(height: 28),
            if (!register) ...[
              field('identifier', 'Email, username ou téléphone'),
              field('password', 'Mot de passe', obscure: true),
            ] else ...[
              Row(
                children: [
                  Expanded(child: field('first_name', 'Prénom')),
                  const SizedBox(width: 12),
                  Expanded(child: field('last_name', 'Nom')),
                ],
              ),
              field('username', 'Username'),
              field('phone', 'Téléphone', type: TextInputType.phone),
              field('email', 'Email', type: TextInputType.emailAddress),
              field('password', 'Mot de passe (8+ caractères)', obscure: true),
              DropdownButtonFormField<String>(
                initialValue: zone,
                decoration: const InputDecoration(labelText: 'Quartier'),
                items: [for (final z in zones) DropdownMenuItem(value: z.name, child: Text(z.name))],
                onChanged: (v) => setState(() => zone = v!),
              ),
              const SizedBox(height: 12),
              Row(
                children: [
                  roleCard(wants, '🙋', 'Je cherche des services', () => setState(() => wants = !wants)),
                  const SizedBox(width: 12),
                  roleCard(offers, '🛠', 'Je propose mes services', () => setState(() => offers = !offers)),
                ],
              ),
              const SizedBox(height: 12),
            ],
            FilledButton(
              onPressed: busy ? null : submit,
              style: FilledButton.styleFrom(padding: const EdgeInsets.all(16)),
              child: Text(register ? 'Créer mon compte' : 'Se connecter'),
            ),
            TextButton(
              onPressed: () => setState(() => register = !register),
              child: Text(register ? 'Déjà inscrit ? Se connecter' : 'Pas de compte ? Créer un compte'),
            ),
          ],
        ),
      ),
    );
  }
}
