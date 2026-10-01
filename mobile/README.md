# MicroJob mobile (Flutter)

Client + worker app for MicroJob. See the [root README](../README.md) for the full picture.

```bash
flutter pub get
flutter run --dart-define=API_URL=http://10.0.2.2:8080   # Android emulator → host API
flutter test && flutter analyze
```

- `lib/api/` — models and REST client
- `lib/state/session.dart` — signed-in user, token storage, realtime WebSocket
- `lib/screens/` — auth, client home, worker home (availability + nearby jobs), post job,
  job detail (all status actions), chat, my jobs / missions, wallet, profile & skills, search
