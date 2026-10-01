import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../state/session.dart';
import 'client_home.dart';
import 'conversations_screen.dart';
import 'lists_screen.dart';
import 'notifications_screen.dart';
import 'post_job_screen.dart';
import 'profile_screen.dart';
import 'worker_home.dart';

/// Bottom-navigation shell. Users who are both client and worker can switch
/// mode from the app bar.
class HomeShell extends StatefulWidget {
  const HomeShell({super.key});

  @override
  State<HomeShell> createState() => _HomeShellState();
}

class _HomeShellState extends State<HomeShell> {
  int tab = 0;
  bool? workerMode;

  @override
  Widget build(BuildContext context) {
    final session = context.watch<Session>();
    final user = session.user!;
    final isWorker = workerMode ?? (user.isWorker && !user.isClient);
    final pages = [
      isWorker && user.isWorker ? const WorkerHome() : const ClientHome(),
      isWorker && user.isWorker ? const MissionsScreen() : const MyJobsScreen(),
      const ConversationsScreen(),
      const ProfileScreen(),
    ];

    return Scaffold(
      appBar: AppBar(
        title: const Text('MicroJob', style: TextStyle(fontWeight: FontWeight.w900)),
        actions: [
          if (user.isWorker && user.isClient)
            TextButton.icon(
              onPressed: () => setState(() => workerMode = !isWorker),
              icon: Icon(isWorker ? Icons.handyman : Icons.person),
              label: Text(isWorker ? 'Worker' : 'Client'),
            ),
          IconButton(
            onPressed: () => Navigator.push(context, MaterialPageRoute(builder: (_) => const NotificationsScreen())),
            icon: Badge(
              isLabelVisible: session.unread > 0,
              label: Text('${session.unread}'),
              child: const Icon(Icons.notifications_outlined),
            ),
          ),
        ],
      ),
      body: IndexedStack(index: tab, children: pages),
      floatingActionButton: tab == 0 && !isWorker
          ? FloatingActionButton.extended(
              onPressed: () => Navigator.push(context, MaterialPageRoute(builder: (_) => const PostJobScreen())),
              icon: const Icon(Icons.add),
              label: const Text('POST A JOB'),
            )
          : null,
      bottomNavigationBar: NavigationBar(
        selectedIndex: tab,
        onDestinationSelected: (i) => setState(() => tab = i),
        destinations: [
          const NavigationDestination(icon: Icon(Icons.home_outlined), label: 'Accueil'),
          NavigationDestination(icon: const Icon(Icons.work_outline), label: isWorker ? 'Missions' : 'Mes jobs'),
          const NavigationDestination(icon: Icon(Icons.chat_bubble_outline), label: 'Messages'),
          const NavigationDestination(icon: Icon(Icons.person_outline), label: 'Profil'),
        ],
      ),
    );
  }
}
