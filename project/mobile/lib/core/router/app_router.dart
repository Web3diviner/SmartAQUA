import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../features/auth/presentation/screens/login_screen.dart';
import '../../features/auth/presentation/screens/register_screen.dart';
import '../../features/auth/presentation/screens/splash_screen.dart';
import '../../features/dashboard/presentation/screens/dashboard_screen.dart';
import '../../features/farm/presentation/screens/farm_management_screen.dart';
import '../../features/farm/presentation/screens/digital_twin_screen.dart';
import '../../features/farm/presentation/screens/farm_simulator_screen.dart';
import '../../features/aquadoc/presentation/screens/aquadoc_chat_screen.dart';
import '../../features/aquadoc/presentation/screens/disease_triage_screen.dart';
import '../../features/settings/presentation/screens/more_hub_screen.dart';
import '../../features/devices/presentation/screens/device_list_screen.dart';
import '../../features/devices/presentation/screens/device_detail_screen.dart';
import '../../features/devices/presentation/screens/device_pairing_screen.dart';
import '../../features/feeding/presentation/screens/feeding_schedule_screen.dart';
import '../../features/feeding/presentation/screens/manual_feed_screen.dart';
import '../../features/feeding/presentation/screens/feeding_history_screen.dart';
import '../../features/monitoring/presentation/screens/monitoring_screen.dart';
import '../../features/calculator/presentation/screens/feed_calculator_screen.dart';
import '../../features/settings/presentation/screens/settings_screen.dart';
import '../../features/video/presentation/screens/video_verification_screen.dart';
import '../../features/onboarding/presentation/screens/onboarding_screen.dart';
import '../providers/auth_provider.dart';

final appRouterProvider = Provider<GoRouter>((ref) {
  final refreshListenable = ValueNotifier<int>(0);
  ref.onDispose(refreshListenable.dispose);
  ref.listen<AuthState>(authStateProvider, (_, _) {
    refreshListenable.value++;
  });

  return GoRouter(
    initialLocation: '/splash',
    debugLogDiagnostics: true,
    refreshListenable: refreshListenable,
    redirect: (context, state) {
      final authState = ref.read(authStateProvider);
      final isLoggedIn = authState.isAuthenticated;
      final isLoggingIn = state.matchedLocation == '/login';
      final isRegistering = state.matchedLocation == '/register';
      final isSplash = state.matchedLocation == '/splash';
      final isOnboarding = state.matchedLocation == '/onboarding';

      if (isSplash || isOnboarding) return null;

      if (!isLoggedIn && !isLoggingIn && !isRegistering) {
        return '/login';
      }

      if (isLoggedIn && (isLoggingIn || isRegistering)) {
        return '/dashboard';
      }

      return null;
    },
    routes: [
      GoRoute(
        path: '/splash',
        name: 'splash',
        builder: (context, state) => const SplashScreen(),
      ),
      GoRoute(
        path: '/onboarding',
        name: 'onboarding',
        builder: (context, state) => const OnboardingScreen(),
      ),
      GoRoute(
        path: '/login',
        name: 'login',
        builder: (context, state) => const LoginScreen(),
      ),
      GoRoute(
        path: '/register',
        name: 'register',
        builder: (context, state) => const RegisterScreen(),
      ),
      ShellRoute(
        builder: (context, state, child) => MainShell(child: child),
        routes: [
          GoRoute(
            path: '/dashboard',
            name: 'dashboard',
            builder: (context, state) => const DashboardScreen(),
          ),
          GoRoute(
            path: '/farm',
            name: 'farm',
            builder: (context, state) => const FarmManagementScreen(),
          ),
          GoRoute(
            path: '/twin',
            name: 'twin',
            builder: (context, state) => const DigitalTwinScreen(),
          ),
          GoRoute(
            path: '/simulator',
            name: 'simulator',
            builder: (context, state) => const FarmSimulatorScreen(),
          ),
          GoRoute(
            path: '/feeding',
            name: 'feeding',
            builder: (context, state) => const FeedingScheduleScreen(),
            routes: [
              GoRoute(
                path: 'manual',
                name: 'manual-feed',
                builder: (context, state) => const ManualFeedScreen(),
              ),
              GoRoute(
                path: 'history',
                name: 'feeding-history',
                builder: (context, state) => const FeedingHistoryScreen(),
              ),
            ],
          ),
          GoRoute(
            path: '/aquadoc',
            name: 'aquadoc',
            builder: (context, state) => const AquaDocChatScreen(),
          ),
          GoRoute(
            path: '/triage',
            name: 'triage',
            builder: (context, state) => const DiseaseTriageScreen(),
          ),
          GoRoute(
            path: '/more',
            name: 'more',
            builder: (context, state) => const MoreHubScreen(),
          ),
          GoRoute(
            path: '/devices',
            name: 'devices',
            builder: (context, state) => const DeviceListScreen(),
            routes: [
              GoRoute(
                path: 'pair',
                name: 'device-pair',
                builder: (context, state) => const DevicePairingScreen(),
              ),
              GoRoute(
                path: ':deviceId',
                name: 'device-detail',
                builder:
                    (context, state) => DeviceDetailScreen(
                      deviceId: state.pathParameters['deviceId']!,
                    ),
              ),
            ],
          ),
          GoRoute(
            path: '/monitoring',
            name: 'monitoring',
            builder: (context, state) => const MonitoringScreen(),
          ),
          GoRoute(
            path: '/video',
            name: 'video',
            builder: (context, state) => const VideoVerificationScreen(),
            routes: [
              GoRoute(
                path: ':feedingEventId',
                name: 'video-verification',
                builder:
                    (context, state) => VideoVerificationScreen(
                      feedingEventId: state.pathParameters['feedingEventId'],
                    ),
              ),
            ],
          ),
          GoRoute(
            path: '/calculator',
            name: 'calculator',
            builder: (context, state) => const FeedCalculatorScreen(),
          ),
          GoRoute(
            path: '/settings',
            name: 'settings',
            builder: (context, state) => const SettingsScreen(),
          ),
        ],
      ),
    ],
  );
});

class MainShell extends StatelessWidget {
  final Widget child;

  const MainShell({super.key, required this.child});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: child,
      bottomNavigationBar: NavigationBar(
        selectedIndex: _calculateSelectedIndex(context),
        onDestinationSelected: (index) => _onItemTapped(index, context),
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.home_outlined),
            selectedIcon: Icon(Icons.home),
            label: 'Home',
          ),
          NavigationDestination(
            icon: Icon(Icons.waves_outlined),
            selectedIcon: Icon(Icons.waves),
            label: 'Farm',
          ),
          NavigationDestination(
            icon: Icon(Icons.restaurant_outlined),
            selectedIcon: Icon(Icons.restaurant),
            label: 'Feeding',
          ),
          NavigationDestination(
            icon: Icon(Icons.psychology_outlined),
            selectedIcon: Icon(Icons.psychology),
            label: 'AquaDoc',
          ),
          NavigationDestination(
            icon: Icon(Icons.grid_view_outlined),
            selectedIcon: Icon(Icons.grid_view),
            label: 'More',
          ),
        ],
      ),
    );
  }

  int _calculateSelectedIndex(BuildContext context) {
    final location = GoRouterState.of(context).matchedLocation;
    if (location.startsWith('/dashboard')) return 0;
    if (location.startsWith('/farm') || location.startsWith('/twin') || location.startsWith('/simulator')) {
      return 1;
    }
    if (location.startsWith('/feeding') || location.startsWith('/calculator')) {
      return 2;
    }
    if (location.startsWith('/aquadoc') || location.startsWith('/triage')) {
      return 3;
    }
    if (location.startsWith('/more') ||
        location.startsWith('/devices') ||
        location.startsWith('/monitoring') ||
        location.startsWith('/video') ||
        location.startsWith('/settings')) {
      return 4;
    }
    return 0;
  }

  void _onItemTapped(int index, BuildContext context) {
    switch (index) {
      case 0:
        context.goNamed('dashboard');
        break;
      case 1:
        context.goNamed('farm');
        break;
      case 2:
        context.goNamed('feeding');
        break;
      case 3:
        context.goNamed('aquadoc');
        break;
      case 4:
        context.goNamed('more');
        break;
    }
  }
}
