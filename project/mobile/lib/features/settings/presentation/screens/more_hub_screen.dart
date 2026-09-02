import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/providers/auth_provider.dart';
import '../../../../core/theme/app_theme.dart';

class MoreHubScreen extends ConsumerWidget {
  const MoreHubScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Precision Hub & Settings'),
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          // Hero Platform Suite Header
          Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              gradient: const LinearGradient(
                colors: [Color(0xFF0F2027), Color(0xFF203A43), Color(0xFF2C5364)],
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
              ),
              borderRadius: BorderRadius.circular(20),
              border: Border.all(color: AppTheme.primaryCyan.withOpacity(0.35)),
              boxShadow: [
                BoxShadow(
                  color: AppTheme.primaryCyan.withOpacity(0.15),
                  blurRadius: 16,
                  offset: const Offset(0, 6),
                ),
              ],
            ),
            child: Row(
              children: [
                Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    gradient: AppTheme.primaryGradient,
                    borderRadius: BorderRadius.circular(16),
                  ),
                  child: const Icon(Icons.hub, color: Colors.white, size: 28),
                ),
                const SizedBox(width: 16),
                const Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'SmartAQUA Precision Suite',
                        style: TextStyle(
                          color: Colors.white,
                          fontSize: 16,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                      SizedBox(height: 2),
                      Text(
                        'Unified Digital Twin & Intelligent Hardware Engine',
                        style: TextStyle(color: Colors.white70, fontSize: 11),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 24),

          // Advanced Intelligence Group
          const _SectionHeader(title: 'AQUADOC CLINICAL AI & DIGITAL TWIN'),
          _HubTile(
            icon: Icons.hub,
            iconColor: AppTheme.primaryCyan,
            title: 'AquaTwin 6-Facet Digital Twin',
            subtitle: 'Live 3D culture tank, sensor streams & 24h timeline scrubber',
            onTap: () => context.go('/twin'),
          ),
          _HubTile(
            icon: Icons.science,
            iconColor: AppTheme.primaryTeal,
            title: 'Farm Environmental Simulator',
            subtitle: 'Multi-factor SGR growth curve, DO hypoxia & TAN stress model',
            onTap: () => context.go('/simulator'),
          ),
          _HubTile(
            icon: Icons.medical_services,
            iconColor: Colors.redAccent,
            title: 'Clinical Disease Triage & Pathology',
            subtitle: 'Diagnostic symptom checklist, confidence ranking & treatment protocols',
            onTap: () => context.go('/triage'),
          ),
          _HubTile(
            icon: Icons.psychology,
            iconColor: Colors.purpleAccent,
            title: 'AquaDoc Clinical AI Advisor',
            subtitle: 'Conversational veterinary consult grounded with hybrid RAG',
            onTap: () => context.go('/aquadoc'),
          ),
          _HubTile(
            icon: Icons.videocam,
            iconColor: Colors.deepPurpleAccent,
            title: 'AquaVision Edge Observation',
            subtitle: 'Surface boiling activity score & feeding response index',
            onTap: () => context.go('/video'),
          ),
          const SizedBox(height: 20),

          // Hardware & Telemetry
          const _SectionHeader(title: 'HARDWARE & TELEMETRY'),
          _HubTile(
            icon: Icons.devices,
            iconColor: Colors.indigoAccent,
            title: 'Fleet Device Management',
            subtitle: 'ESP32 automated feeders, load cells & sensor nodes',
            onTap: () => context.go('/devices'),
          ),
          _HubTile(
            icon: Icons.bluetooth_searching,
            iconColor: Colors.blueAccent,
            title: 'BLE Device Provisioning',
            subtitle: 'Pair new feeders & calibrate load cells via Bluetooth',
            onTap: () => context.go('/devices/pair'),
          ),
          _HubTile(
            icon: Icons.monitor_heart,
            iconColor: Colors.cyanAccent,
            title: 'Multisensor Diagnostics & Alerts',
            subtitle: 'Telemetry streams for DO, pH, Temp, TAN and system health',
            onTap: () => context.go('/monitoring'),
          ),
          _HubTile(
            icon: Icons.calculate,
            iconColor: Colors.tealAccent,
            title: 'Feed Conversion & Nutrition Calculator',
            subtitle: 'Calculate daily rations based on pond temperature & biomass',
            onTap: () => context.go('/calculator'),
          ),
          const SizedBox(height: 20),

          // Research Export
          const _SectionHeader(title: 'RESEARCH DATA EXPORT'),
          _HubTile(
            icon: Icons.file_download,
            iconColor: Colors.greenAccent,
            title: 'Download Research Bundle (JSON / CSV)',
            subtitle: 'Export time-series datasets, feeding events, and biometrics',
            onTap: () {
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(
                  content: Text('Research export generated: 14-day telemetry & feeding CSV bundle ready.'),
                  backgroundColor: AppTheme.deviceOnline,
                ),
              );
            },
          ),
          const SizedBox(height: 20),

          // Settings & Account
          const _SectionHeader(title: 'PREFERENCES & SYSTEM'),
          _HubTile(
            icon: Icons.settings,
            iconColor: Colors.grey,
            title: 'App Settings & Units',
            subtitle: 'Notifications, metric/imperial units, and theme',
            onTap: () => context.go('/settings'),
          ),
          _HubTile(
            icon: Icons.logout,
            iconColor: Colors.red,
            title: 'Sign Out / Switch Farm',
            subtitle: 'Exit current sandbox session',
            onTap: () async {
              await ref.read(authStateProvider.notifier).logout();
              if (context.mounted) {
                context.go('/login');
              }
            },
          ),
          const SizedBox(height: 32),
        ],
      ),
    );
  }
}

class _SectionHeader extends StatelessWidget {
  final String title;

  const _SectionHeader({required this.title});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(left: 4, bottom: 8),
      child: Text(
        title,
        style: TextStyle(
          fontSize: 11,
          fontWeight: FontWeight.bold,
          letterSpacing: 1.1,
          color: Colors.grey[500],
        ),
      ),
    );
  }
}

class _HubTile extends StatelessWidget {
  final IconData icon;
  final Color iconColor;
  final String title;
  final String subtitle;
  final VoidCallback onTap;

  const _HubTile({
    required this.icon,
    required this.iconColor,
    required this.title,
    required this.subtitle,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      child: ListTile(
        onTap: onTap,
        contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
        leading: Container(
          padding: const EdgeInsets.all(10),
          decoration: BoxDecoration(
            color: iconColor.withOpacity(0.12),
            borderRadius: BorderRadius.circular(12),
            border: Border.all(color: iconColor.withOpacity(0.25)),
          ),
          child: Icon(icon, color: iconColor, size: 22),
        ),
        title: Text(
          title,
          style: const TextStyle(fontSize: 14, fontWeight: FontWeight.bold),
        ),
        subtitle: Text(
          subtitle,
          style: TextStyle(fontSize: 12, color: Colors.grey[500]),
        ),
        trailing: const Icon(Icons.chevron_right, size: 20, color: Colors.grey),
      ),
    );
  }
}
