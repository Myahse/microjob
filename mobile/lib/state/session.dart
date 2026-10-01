import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import '../api/api_client.dart';
import '../api/models.dart';

/// A realtime event pushed by the API over WebSocket.
class RealtimeEvent {
  final String type; // notification | message | job
  final Map<String, dynamic> data;
  RealtimeEvent(this.type, this.data);
}

/// Holds the signed-in user, the API client and the realtime connection.
class Session extends ChangeNotifier {
  Session(this.api) {
    api.onUnauthorized = signOut;
  }

  static const _tokenKey = 'microjob.token';

  final ApiClient api;
  User? user;
  bool loading = true;
  int unread = 0;

  final _events = StreamController<RealtimeEvent>.broadcast();
  Stream<RealtimeEvent> get events => _events.stream;
  WebSocketChannel? _ws;
  StreamSubscription? _wsSub;
  Timer? _reconnect;

  Future<void> restore() async {
    final prefs = await SharedPreferences.getInstance();
    api.token = prefs.getString(_tokenKey);
    if (api.token != null) {
      try {
        user = await api.me();
        await refreshUnread();
        _connect();
      } catch (_) {
        api.token = null;
        await prefs.remove(_tokenKey);
      }
    }
    loading = false;
    notifyListeners();
  }

  Future<void> signIn(String token, User u) async {
    api.token = token;
    user = u;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_tokenKey, token);
    _connect();
    await refreshUnread();
    notifyListeners();
  }

  Future<void> signOut() async {
    _disconnect();
    api.token = null;
    user = null;
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_tokenKey);
    notifyListeners();
  }

  void setUser(User u) {
    user = u;
    notifyListeners();
  }

  Future<void> refreshUser() async {
    user = await api.me();
    notifyListeners();
  }

  Future<void> refreshUnread() async {
    try {
      final (_, n) = await api.notifications();
      unread = n;
      notifyListeners();
    } catch (_) {}
  }

  void clearUnread() {
    unread = 0;
    notifyListeners();
  }

  void _connect() {
    _disconnect();
    if (api.token == null) return;
    try {
      _ws = WebSocketChannel.connect(api.websocketUri());
      _wsSub = _ws!.stream.listen(
        (raw) {
          final m = jsonDecode(raw as String) as Map<String, dynamic>;
          final e = RealtimeEvent(m['type'] as String, (m['data'] as Map).cast<String, dynamic>());
          if (e.type == 'notification') {
            unread++;
            notifyListeners();
          }
          _events.add(e);
        },
        onDone: _scheduleReconnect,
        onError: (_) => _scheduleReconnect(),
        cancelOnError: true,
      );
    } catch (_) {
      _scheduleReconnect();
    }
  }

  void _scheduleReconnect() {
    if (api.token == null) return;
    _reconnect?.cancel();
    _reconnect = Timer(const Duration(seconds: 3), _connect);
  }

  void _disconnect() {
    _reconnect?.cancel();
    _wsSub?.cancel();
    _ws?.sink.close();
    _ws = null;
  }

  @override
  void dispose() {
    _disconnect();
    _events.close();
    super.dispose();
  }
}
