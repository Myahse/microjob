import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../api/models.dart';
import '../state/session.dart';
import '../util/format.dart';

class WalletScreen extends StatefulWidget {
  const WalletScreen({super.key});

  @override
  State<WalletScreen> createState() => _WalletScreenState();
}

class _WalletScreenState extends State<WalletScreen> {
  Wallet? wallet;

  @override
  void initState() {
    super.initState();
    load();
  }

  Future<void> load() async {
    try {
      final w = await context.read<Session>().api.wallet();
      if (mounted) setState(() => wallet = w);
    } catch (e) {
      if (mounted) toast(context, e);
    }
  }

  Future<void> withdraw() async {
    final api = context.read<Session>().api;
    final amount = TextEditingController();
    final dest = TextEditingController(text: context.read<Session>().user!.phone);
    var method = 'mobile_money';
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, set) => AlertDialog(
          title: const Text('Demander un retrait'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: amount,
                keyboardType: TextInputType.number,
                decoration: const InputDecoration(labelText: 'Montant (FCFA)'),
              ),
              const SizedBox(height: 8),
              SegmentedButton<String>(
                segments: const [
                  ButtonSegment(value: 'mobile_money', label: Text('Mobile Money')),
                  ButtonSegment(value: 'bank', label: Text('Banque')),
                ],
                selected: {method},
                onSelectionChanged: (v) => set(() => method = v.first),
              ),
              const SizedBox(height: 8),
              TextField(
                controller: dest,
                decoration: InputDecoration(labelText: method == 'bank' ? 'IBAN / RIB' : 'Numéro'),
              ),
            ],
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Annuler')),
            FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('Retirer')),
          ],
        ),
      ),
    );
    if (ok != true) return;
    try {
      final w = await api.withdraw(int.tryParse(amount.text) ?? 0, method, dest.text);
      setState(() => wallet = w);
      if (mounted) toast(context, 'Retrait demandé');
    } catch (e) {
      if (mounted) toast(context, e);
    }
  }

  @override
  Widget build(BuildContext context) {
    final w = wallet;
    return Scaffold(
      appBar: AppBar(title: const Text('Revenus')),
      body: w == null
          ? const Center(child: CircularProgressIndicator())
          : RefreshIndicator(
              onRefresh: load,
              child: ListView(
                padding: const EdgeInsets.all(12),
                children: [
                  Card(
                    color: Colors.indigo,
                    child: Padding(
                      padding: const EdgeInsets.all(20),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          const Text('Solde disponible', style: TextStyle(color: Colors.white70)),
                          Text(
                            fcfa(w.balance),
                            style: const TextStyle(color: Colors.white, fontSize: 28, fontWeight: FontWeight.w800),
                          ),
                          const SizedBox(height: 8),
                          Text(
                            'Ce mois-ci : ${fcfa(w.earnedThisMonth)} · Total : ${fcfa(w.totalEarned)}',
                            style: const TextStyle(color: Colors.white),
                          ),
                          if (w.pendingWithdrawals > 0)
                            Text(
                              '${fcfa(w.pendingWithdrawals)} en cours de retrait',
                              style: const TextStyle(color: Colors.white70),
                            ),
                          const SizedBox(height: 12),
                          FilledButton.tonal(onPressed: w.balance > 0 ? withdraw : null, child: const Text('Retirer')),
                        ],
                      ),
                    ),
                  ),
                  const Padding(
                    padding: EdgeInsets.all(8),
                    child: Text('Historique', style: TextStyle(fontWeight: FontWeight.bold)),
                  ),
                  if (w.transactions.isEmpty) const Padding(padding: EdgeInsets.all(16), child: Text('Aucune transaction.')),
                  for (final t in w.transactions)
                    ListTile(
                      title: Text(t.description),
                      subtitle: Text(ago(t.createdAt)),
                      trailing: Text(
                        '${t.amount >= 0 ? '+' : ''}${fcfa(t.amount)}',
                        style: TextStyle(fontWeight: FontWeight.bold, color: t.amount >= 0 ? Colors.green : Colors.red),
                      ),
                    ),
                ],
              ),
            ),
    );
  }
}
