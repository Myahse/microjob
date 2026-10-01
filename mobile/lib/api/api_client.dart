import 'dart:convert';

import 'package:http/http.dart' as http;

import '../config.dart';
import 'models.dart';

class ApiException implements Exception {
  final int status;
  final String message;
  ApiException(this.status, this.message);

  @override
  String toString() => message;
}

/// Thin typed client over the MicroJob REST API.
class ApiClient {
  ApiClient({this.baseUrl = apiUrl, http.Client? client}) : _http = client ?? http.Client();

  final String baseUrl;
  final http.Client _http;
  String? token;
  void Function()? onUnauthorized;

  String media(String url) => url.startsWith('/') ? '$baseUrl$url' : url;

  Future<dynamic> _send(String method, String path, [Object? body]) async {
    final req = http.Request(method, Uri.parse('$baseUrl$path'));
    if (token != null) req.headers['Authorization'] = 'Bearer $token';
    if (body != null) {
      req.headers['Content-Type'] = 'application/json';
      req.body = jsonEncode(body);
    }
    final res = await http.Response.fromStream(await _http.send(req));
    if (res.statusCode == 401 && token != null) onUnauthorized?.call();
    if (res.statusCode == 204 || res.body.isEmpty) return null;
    final data = jsonDecode(utf8.decode(res.bodyBytes));
    if (res.statusCode >= 400) {
      throw ApiException(res.statusCode, (data is Map ? data['error'] : null) as String? ?? 'Erreur ${res.statusCode}');
    }
    return data;
  }

  Future<dynamic> _get(String p) => _send('GET', p);
  Future<dynamic> _post(String p, [Object? b]) => _send('POST', p, b);

  // ---- auth & profile ----
  Future<(String, User)> login(String identifier, String password) async {
    final d = await _post('/api/auth/login', {'identifier': identifier, 'password': password});
    return (d['token'] as String, User.fromJson(d['user']));
  }

  Future<(String, User)> register(Map<String, dynamic> body) async {
    final d = await _post('/api/auth/register', body);
    return (d['token'] as String, User.fromJson(d['user']));
  }

  Future<User> me() async => User.fromJson(await _get('/api/me'));
  Future<User> updateMe(Map<String, dynamic> b) async => User.fromJson(await _send('PATCH', '/api/me', b));
  Future<User> updateWorker(Map<String, dynamic> b) async => User.fromJson(await _send('PUT', '/api/me/worker', b));
  Future<User> setAvailability(bool available, {double? lat, double? lng, String? zone}) async => User.fromJson(
    await _send('PUT', '/api/me/availability', {
      'available': available,
      'latitude': ?lat,
      'longitude': ?lng,
      'zone': ?zone,
    }),
  );

  Future<(User, List<Review>)> profile(String id) async {
    final d = await _get('/api/users/$id');
    return (User.fromJson(d['user']), (d['reviews'] as List).map((e) => Review.fromJson(e)).toList());
  }

  Future<List<Category>> categories() async => (await _get('/api/categories') as List).map((e) => Category.fromJson(e)).toList();

  Future<List<WorkerResult>> searchWorkers(String q, {double? lat, double? lng}) async {
    final params = {'q': q, if (lat != null) 'lat': '$lat', if (lng != null) 'lng': '$lng'};
    final d = await _get('/api/workers/search?${Uri(queryParameters: params).query}') as List;
    return d.map((e) => WorkerResult.fromJson(e)).toList();
  }

  Future<int> nearbyCount(double lat, double lng, {double radiusKm = 5}) async =>
      (await _get('/api/workers/nearby-count?lat=$lat&lng=$lng&radius_km=$radiusKm'))['count'] as int;

  // ---- jobs ----
  List<Job> _jobs(dynamic d) => (d as List).map((e) => Job.fromJson(e)).toList();

  Future<(Job, int)> createJob(Map<String, dynamic> body) async {
    final d = await _post('/api/jobs', body);
    return (Job.fromJson(d['job']), d['notified_workers'] as int);
  }

  Future<List<Job>> myJobs(String scope) async => _jobs(await _get('/api/jobs/mine?scope=$scope'));
  Future<List<Job>> missions(String scope) async => _jobs(await _get('/api/jobs/missions?scope=$scope'));
  Future<List<Job>> availableJobs() async => _jobs(await _get('/api/jobs/available'));
  Future<Job> job(String id) async => Job.fromJson(await _get('/api/jobs/$id'));
  Future<Job> accept(String id) async => Job.fromJson(await _post('/api/jobs/$id/accept'));
  Future<void> decline(String id) => _post('/api/jobs/$id/decline');
  Future<List<Application>> applications(String id) async =>
      (await _get('/api/jobs/$id/applications') as List).map((e) => Application.fromJson(e)).toList();
  Future<Job> apply(String id, int price, String message) async =>
      Job.fromJson(await _post('/api/jobs/$id/applications', {'price': price, 'message': message}));
  Future<Job> choose(String id, String appId) async => Job.fromJson(await _post('/api/jobs/$id/applications/$appId/choose'));
  Future<Job> advance(String id, String status) async => Job.fromJson(await _post('/api/jobs/$id/status', {'status': status}));
  Future<Job> confirm(String id) async => Job.fromJson(await _post('/api/jobs/$id/confirm'));
  Future<Job> cancel(String id, String reason) async => Job.fromJson(await _post('/api/jobs/$id/cancel', {'reason': reason}));
  Future<void> shareLocation(String id, double lat, double lng) =>
      _send('PUT', '/api/jobs/$id/location', {'latitude': lat, 'longitude': lng});
  Future<Job> review(String id, Map<String, dynamic> body) async => Job.fromJson(await _post('/api/jobs/$id/reviews', body));
  Future<Job> report(String id, String type, String description) async =>
      Job.fromJson(await _post('/api/jobs/$id/reports', {'type': type, 'description': description}));

  // ---- chat ----
  Future<List<Message>> messages(String jobId) async =>
      (await _get('/api/jobs/$jobId/messages') as List).map((e) => Message.fromJson(e)).toList();
  Future<Message> sendMessage(String jobId, Map<String, dynamic> body) async =>
      Message.fromJson(await _post('/api/jobs/$jobId/messages', body));
  Future<List<Conversation>> conversations() async =>
      (await _get('/api/conversations') as List).map((e) => Conversation.fromJson(e)).toList();

  // ---- wallet, notifications, uploads ----
  Future<Wallet> wallet() async => Wallet.fromJson(await _get('/api/wallet'));
  Future<Wallet> withdraw(int amount, String method, String destination) async =>
      Wallet.fromJson(await _post('/api/wallet/withdrawals', {'amount': amount, 'method': method, 'destination': destination}));

  Future<(List<AppNotification>, int)> notifications() async {
    final d = await _get('/api/notifications');
    return ((d['items'] as List).map((e) => AppNotification.fromJson(e)).toList(), d['unread'] as int);
  }

  Future<void> markAllRead() => _post('/api/notifications/read', {'ids': <String>[]});

  Future<String> upload(List<int> bytes, String filename) async {
    final req = http.MultipartRequest('POST', Uri.parse('$baseUrl/api/uploads'));
    if (token != null) req.headers['Authorization'] = 'Bearer $token';
    req.files.add(http.MultipartFile.fromBytes('file', bytes, filename: filename));
    final res = await http.Response.fromStream(await _http.send(req));
    final data = jsonDecode(res.body);
    if (res.statusCode >= 400) throw ApiException(res.statusCode, data['error'] as String? ?? 'Upload échoué');
    return data['url'] as String;
  }

  Uri websocketUri() {
    final base = Uri.parse(baseUrl);
    return base.replace(scheme: base.scheme == 'https' ? 'wss' : 'ws', path: '/api/ws', queryParameters: {'token': token ?? ''});
  }
}
