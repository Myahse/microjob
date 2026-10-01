import 'package:flutter_test/flutter_test.dart';
import 'package:microjob/api/models.dart';
import 'package:microjob/util/format.dart';

void main() {
  test('parses a job as seen by a candidate worker', () {
    final job = Job.fromJson({
      'id': 'j1',
      'title': 'Installer Windows',
      'description': '',
      'category': {'id': 1, 'name': 'Tech', 'icon': '💻'},
      'skill_id': 1,
      'skill_name': 'Installation Windows',
      'zone': 'Cocody',
      'pricing_mode': 'fixed',
      'budget': 10000,
      'agreed_price': null,
      'scheduled_at': null,
      'duration_minutes': 120,
      'photos': [],
      'status': 'MATCHING',
      'created_at': '2026-10-01T10:00:00Z',
      'updated_at': '2026-10-01T10:00:00Z',
      'client': {'id': 'c1', 'first_name': 'Aïcha', 'last_name': 'Y', 'avatar_url': '', 'rating': 0, 'rating_count': 0},
      'worker': null,
      'applications_count': 0,
      'my_role': 'candidate',
      'distance_km': 1.8,
      'reviewed_by_me': false,
      'reviewed_by_other': false,
      'open_report': false,
      'has_conversation': false,
    });
    expect(job.isOpen, isTrue);
    expect(job.price, 10000);
    expect(job.address, isNull);
    expect(job.distanceKm, 1.8);
  });

  test('formats amounts and distances', () {
    expect(fcfa(10000), '10 000 FCFA');
    expect(km(1.84), '1.8 km');
    expect(statusLabel['AWAITING_CONFIRMATION'], 'À confirmer');
  });
}
