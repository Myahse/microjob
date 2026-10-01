import 'package:flutter/material.dart';
import 'package:geolocator/geolocator.dart';
import 'package:intl/intl.dart';

final _fcfa = NumberFormat.decimalPattern('fr_FR');

String fcfa(int n) => '${_fcfa.format(n).replaceAll(' ', ' ')} FCFA';

String km(double? d) => d == null ? '' : '${d.toStringAsFixed(1)} km';

String when(DateTime? d) {
  if (d == null) return 'Dès que possible';
  final now = DateTime.now();
  final time = DateFormat.Hm().format(d);
  if (DateUtils.isSameDay(d, now)) return "Aujourd'hui $time";
  if (DateUtils.isSameDay(d, now.add(const Duration(days: 1)))) return 'Demain $time';
  return '${DateFormat('dd/MM').format(d)} $time';
}

String ago(DateTime d) {
  final s = DateTime.now().difference(d).inSeconds;
  if (s < 60) return "à l'instant";
  if (s < 3600) return 'il y a ${s ~/ 60} min';
  if (s < 86400) return 'il y a ${s ~/ 3600} h';
  return DateFormat('dd/MM').format(d);
}

const statusLabel = {
  'POSTED': 'Publié',
  'MATCHING': 'Recherche de worker',
  'ASSIGNED': 'Worker trouvé',
  'WORKER_ON_THE_WAY': 'Worker en route',
  'ARRIVED': 'Worker arrivé',
  'STARTED': 'En cours',
  'AWAITING_CONFIRMATION': 'À confirmer',
  'COMPLETED': 'Terminé',
  'PAID': 'Payé',
  'RATED': 'Évalué',
  'CANCELLED': 'Annulé',
  'DISPUTED': 'Litige',
};

Color statusColor(String s) {
  switch (s) {
    case 'MATCHING':
    case 'POSTED':
      return Colors.amber.shade700;
    case 'AWAITING_CONFIRMATION':
      return Colors.orange.shade700;
    case 'COMPLETED':
    case 'PAID':
    case 'RATED':
      return Colors.green.shade700;
    case 'CANCELLED':
    case 'DISPUTED':
      return Colors.red.shade700;
    default:
      return Colors.indigo;
  }
}

const reportTypes = {
  'not_done': 'Travail non réalisé',
  'bad_quality': 'Mauvaise qualité',
  'payment': 'Paiement',
  'behavior': 'Comportement',
  'fraud': 'Fraude',
  'other': 'Autre',
};

class Zone {
  final String name;
  final double lat;
  final double lng;
  const Zone(this.name, this.lat, this.lng);
}

/// Launch zone: Cocody neighbourhoods.
const zones = [
  Zone('Cocody Centre', 5.348, -3.987),
  Zone('Riviera', 5.36, -3.97),
  Zone('Riviera Palmeraie', 5.372, -3.945),
  Zone('Angré', 5.395, -3.99),
  Zone('II Plateaux', 5.37, -3.995),
  Zone('Blockhaus', 5.333, -3.993),
  Zone('Danga', 5.338, -4.003),
  Zone('Bonoumin', 5.367, -3.958),
];

Zone zoneNamed(String name) => zones.firstWhere((z) => z.name == name, orElse: () => zones.first);

/// Current GPS position, asking for permission if needed.
Future<Position> currentPosition() async {
  if (!await Geolocator.isLocationServiceEnabled()) {
    throw Exception('Activez la localisation');
  }
  var perm = await Geolocator.checkPermission();
  if (perm == LocationPermission.denied) perm = await Geolocator.requestPermission();
  if (perm == LocationPermission.denied || perm == LocationPermission.deniedForever) {
    throw Exception('Permission de localisation refusée');
  }
  return Geolocator.getCurrentPosition();
}

void toast(BuildContext context, Object message) {
  ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(message.toString().replaceFirst('Exception: ', ''))));
}
