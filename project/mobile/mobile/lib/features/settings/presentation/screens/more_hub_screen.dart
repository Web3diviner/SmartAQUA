import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import '../../../core/constants/app_colors.dart';

class MoreHubScreen extends StatelessWidget {
  const MoreHubScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Precision Hub', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
            Text('Vision, Predictions, Research & Settings', style: TextStyle(fontSize: 11, color: Colors.grey)),
          ],
        ),
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          // Section 1: AI & Advanced Vision
          const Text('INTELLIGENCE & COMPUTER VISION', style: TextStyle(fontSize: 11, fontWeight: FontWeight.bold, color: Colors.grey, letterSpacing: 1.0)),
          const SizedBox(height: 8),
          _buildHubTile(
            icon: Icons.videocam_outlined,
            color: Colors.purple,
            title: 'AquaVision Edge Feeds',
            subtitle: 'Real-time surface activity, boil index & fish response',
            onTap: () => context.goNamed('video'),
          ),
          _buildHubTile(
            icon: Icons.trending_up,
            color: Colors.indigo,
            title: 'AquaPredict Growth & Harvest',
            subtitle: 'Biometric trajectories, days to harvest & FCR projections',
            onTap: () => context.goNamed('calculator'),
          ),
          _buildHubTile(
            icon: Icons.warning_amber_outlined,
            color: Colors.amber[800]!,
            title: 'Unified Safety & Alert Interlocks',
            subtitle: 'Hypoxia blocks, ammonia warnings & active alerts',
            onTap: () => context.goNamed('monitoring'),
          ),

          const SizedBox(height: 20),
          // Section 2: Hardware & Infrastructure
          const Text('HARDWARE & TELEMETRY', style: TextStyle(fontSize: 11, fontWeight: FontWeight.bold, color: Colors.grey, letterSpacing: 1.0)),
          const SizedBox(height: 8),
          _buildHubTile(
            icon: Icons.devices_outlined,
            color: Colors.blue,
            title: 'Device Fleet Management',
            subtitle: 'Automatic feeders, sensors, ESP32 BLE provisioning',
            onTap: () => context.goNamed('devices'),
          ),

          const SizedBox(height: 20),
          // Section 3: Research & Data Export
          const Text('RESEARCH & SCIENTIFIC OBSERVABILITY', style: TextStyle(fontSize: 11, fontWeight: FontWeight.bold, color: Colors.grey, letterSpacing: 1.0)),
          const SizedBox(height: 8),
          _buildHubTile(
            icon: Icons.download_outlined,
            color: Colors.teal,
            title: 'Research Data Export',
            subtitle: 'Export precision datasets (JSON / CSV) for academic research',
            onTap: () {
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(content: Text('Datasets ready for download via Go Core API (/api/v1/export/research)')),
              );
            },
          ),

          const SizedBox(height: 20),
          // Section 4: System Settings
          const Text('SYSTEM PREFERENCES', style: TextStyle(fontSize: 11, fontWeight: FontWeight.bold, color: Colors.grey, letterSpacing: 1.0)),
          const SizedBox(height: 8),
          _buildHubTile(
            icon: Icons.settings_outlined,
            color: Colors.blueGrey,
            title: 'Settings & Security',
            subtitle: 'MTLS certificates, notification boundaries & profiles',
            onTap: () => context.goNamed('settings'),
          ),
        ],
      ),
    );
  }

  Widget _buildHubTile({
    required IconData icon,
    required Color color,
    required String title,
    required String subtitle,
    required VoidCallback onTap,
  }) {
    return Card(
      margin: const EdgeInsets.only(bottom: 10),
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: Colors.grey.withOpacity(0.2)),
      ),
      child: ListTile(
        leading: Container(
          padding: const EdgeInsets.all(8),
          decoration: BoxDecoration(
            color: color.withOpacity(0.12),
            borderRadius: BorderRadius.circular(10),
          ),
          child: Icon(icon, color: color, size: 22),
        ),
        title: Text(title, style: const TextStyle(fontSize: 14, fontWeight: FontWeight.bold)),
        subtitle: Text(subtitle, style: TextStyle(fontSize: 11, color: Colors.grey[600])),
        trailing: const Icon(Icons.chevron_right, color: Colors.grey, size: 20),
        onTap: onTap,
      ),
    );
  }
}
