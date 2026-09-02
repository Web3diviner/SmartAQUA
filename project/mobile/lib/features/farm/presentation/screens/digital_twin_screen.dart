import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/theme/app_theme.dart';
import '../widgets/digital_twin_3d_visualizer.dart';

class DigitalTwinScreen extends ConsumerStatefulWidget {
  const DigitalTwinScreen({super.key});

  @override
  ConsumerState<DigitalTwinScreen> createState() => _DigitalTwinScreenState();
}

class _DigitalTwinScreenState extends ConsumerState<DigitalTwinScreen> {
  double _timelineHour = 14.0; // Current time: 14:00 (2:00 PM)
  String _selectedUnit = 'Earthen Pond 1';

  // Digital Twin Facets calculated based on timeline hour
  double get _calcDO => (5.8 + 0.8 * (1 - ((_timelineHour - 14).abs() / 12))).clamp(2.8, 7.2);
  double get _calcTemp => (28.4 + 1.2 * (1 - ((_timelineHour - 15).abs() / 12))).clamp(25.0, 31.0);
  double get _calcTAN => (0.15 + (_timelineHour >= 12 && _timelineHour <= 18 ? 0.05 : 0.0)).clamp(0.05, 0.4);
  double get _calcSurfaceBoil => (_timelineHour == 8 || _timelineHour == 13 || _timelineHour == 17) ? 82.0 : 24.0;
  bool get _isInterlockActive => _calcDO < 3.0;

  @override
  Widget build(BuildContext context) {
    final doValue = _calcDO;
    final tempValue = _calcTemp;
    final tanValue = _calcTAN;
    final boilScore = _calcSurfaceBoil;
    final isBlocked = _isInterlockActive;

    return Scaffold(
      appBar: AppBar(
        title: const Row(
          children: [
            Icon(Icons.hub, color: Colors.blueAccent),
            SizedBox(width: 8),
            Text('AquaTwin 6-Facet Digital Twin'),
          ],
        ),
        actions: [
          IconButton(
            icon: const Icon(Icons.science_outlined),
            tooltip: 'Open Farm Simulator',
            onPressed: () => context.go('/simulator'),
          ),
          IconButton(
            icon: const Icon(Icons.psychology_outlined),
            tooltip: 'Consult AquaDoc AI',
            onPressed: () => context.go('/aquadoc'),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          // Pond Selector & State Header
          Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              gradient: LinearGradient(
                colors: isBlocked
                    ? [Colors.red[900]!, Colors.red[800]!]
                    : [const Color(0xFF0F2027), const Color(0xFF203A43), const Color(0xFF2C5364)],
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
              ),
              borderRadius: BorderRadius.circular(20),
              boxShadow: [
                BoxShadow(
                  color: Colors.black.withOpacity(0.2),
                  blurRadius: 10,
                  offset: const Offset(0, 4),
                ),
              ],
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    DropdownButtonHideUnderline(
                      child: DropdownButton<String>(
                        value: _selectedUnit,
                        dropdownColor: const Color(0xFF203A43),
                        style: const TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 16),
                        icon: const Icon(Icons.arrow_drop_down, color: Colors.white),
                        items: ['Earthen Pond 1', 'Concrete Tank 2', 'RAS Tank Alpha']
                            .map((u) => DropdownMenuItem(value: u, child: Text(u)))
                            .toList(),
                        onChanged: (val) {
                          if (val != null) setState(() => _selectedUnit = val);
                        },
                      ),
                    ),
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                      decoration: BoxDecoration(
                        color: isBlocked ? Colors.red : Colors.green.withOpacity(0.25),
                        borderRadius: BorderRadius.circular(12),
                        border: Border.all(color: isBlocked ? Colors.white : Colors.greenAccent),
                      ),
                      child: Text(
                        isBlocked ? 'SAFETY INTERLOCK ACTIVE' : 'TWIN STATE: OPTIMAL',
                        style: TextStyle(
                          color: isBlocked ? Colors.white : Colors.greenAccent,
                          fontSize: 10,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    const Icon(Icons.access_time, color: Colors.white70, size: 16),
                    const SizedBox(width: 6),
                    Text(
                      'Timeline Snapshot: ${_timelineHour.toInt().toString().padLeft(2, '0')}:00 hrs',
                      style: const TextStyle(color: Colors.white70, fontSize: 13),
                    ),
                    const Spacer(),
                    Text(
                      'Biomass: 1,552 kg',
                      style: const TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 13),
                    ),
                  ],
                ),
                const SizedBox(height: 12),

                // 24-Hour Timeline Scrubber Slider
                SliderTheme(
                  data: SliderTheme.of(context).copyWith(
                    activeTrackColor: Theme.of(context).colorScheme.primary,
                    inactiveTrackColor: Colors.white24,
                    thumbColor: Colors.white,
                    trackHeight: 4,
                  ),
                  child: Slider(
                    value: _timelineHour,
                    min: 0.0,
                    max: 23.0,
                    divisions: 23,
                    label: '${_timelineHour.toInt()}:00',
                    onChanged: (val) => setState(() => _timelineHour = val),
                  ),
                ),
                const Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Text('00:00 (Night)', style: TextStyle(color: Colors.white38, fontSize: 10)),
                    Text('06:00 (Dawn DO Low)', style: TextStyle(color: Colors.white38, fontSize: 10)),
                    Text('12:00 (Noon)', style: TextStyle(color: Colors.white38, fontSize: 10)),
                    Text('18:00 (Dusk)', style: TextStyle(color: Colors.white38, fontSize: 10)),
                    Text('23:00', style: TextStyle(color: Colors.white38, fontSize: 10)),
                  ],
                ),
              ],
            ),
          ),
          const SizedBox(height: 20),

          // Interlock Alert Banner if active
          if (isBlocked) ...[
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: Colors.red.withOpacity(0.12),
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: Colors.red),
              ),
              child: const Row(
                children: [
                  Icon(Icons.dangerous, color: Colors.red),
                  SizedBox(width: 10),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text('Deterministic Feed Interlock Triggered', style: TextStyle(fontWeight: FontWeight.bold, color: Colors.red)),
                        Text('DO < 3.0 mg/L. Physical feeding is blocked to prevent fish mortality.', style: TextStyle(fontSize: 12)),
                      ],
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 16),
          ],

          // 3D Animated Culture Tank Visualizer
          DigitalTwin3DVisualizer(
            dissolvedOxygen: doValue,
            temperature: tempValue,
            ammoniaTan: tanValue,
            avgWeightG: 320.0,
            biomassKg: 1552.0,
            population: 4850,
          ),
          const SizedBox(height: 20),

          Text(
            'The 6 Facets of the Digital Twin',
            style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 12),

          // FACET 1: Environment
          _FacetCard(
            facetNumber: '1',
            facetName: 'Environmental Facet',
            icon: Icons.thermostat,
            iconColor: Colors.blue,
            metrics: [
              _MetricItem(label: 'Dissolved Oxygen', value: '${doValue.toStringAsFixed(1)} mg/L', isAlert: doValue < 3.0),
              _MetricItem(label: 'Water Temperature', value: '${tempValue.toStringAsFixed(1)} °C', isAlert: tempValue < 20 || tempValue > 34),
              _MetricItem(label: 'pH Level', value: '7.4', isAlert: false),
              _MetricItem(label: 'TAN Ammonia', value: '${tanValue.toStringAsFixed(2)} mg/L', isAlert: tanValue > 1.0),
              _MetricItem(label: 'DO Saturation', value: '${(doValue / 7.8 * 100).toInt()}%', isAlert: false),
              _MetricItem(label: 'Nitrite (NO2)', value: '0.02 mg/L', isAlert: false),
            ],
          ),
          const SizedBox(height: 12),

          // FACET 2: Biological
          _FacetCard(
            facetNumber: '2',
            facetName: 'Biological & Fish Facet',
            icon: Icons.set_meal,
            iconColor: Colors.teal,
            metrics: [
              const _MetricItem(label: 'Species', value: 'Clarias gariepinus'),
              const _MetricItem(label: 'Population', value: '4,850 fish'),
              const _MetricItem(label: 'Average Weight', value: '320 g'),
              const _MetricItem(label: 'Total Biomass', value: '1,552 kg'),
              const _MetricItem(label: 'Specific Growth Rate', value: '2.45 %/day'),
              const _MetricItem(label: 'Survival Rate', value: '97.4 %'),
            ],
          ),
          const SizedBox(height: 12),

          // FACET 3: Feeding
          _FacetCard(
            facetNumber: '3',
            facetName: 'Feeding Automation Facet',
            icon: Icons.restaurant,
            iconColor: Colors.orange,
            metrics: [
              const _MetricItem(label: 'Dispensed Today', value: '550 g'),
              const _MetricItem(label: 'Daily Target', value: '800 g (3 rations)'),
              const _MetricItem(label: 'Cumulative FCR', value: '1.18'),
              _MetricItem(label: 'Q10 Factor', value: '${(1.0 + (tempValue - 28.0) * 0.05).clamp(0.8, 1.2).toStringAsFixed(2)}x'),
              _MetricItem(label: 'Interlock Status', value: isBlocked ? 'LOCKED (NO FEED)' : 'UNLOCKED (SAFE)', isAlert: isBlocked),
              const _MetricItem(label: 'Next Feed Event', value: '17:00 (250g)'),
            ],
          ),
          const SizedBox(height: 12),

          // FACET 4: Equipment
          const _FacetCard(
            facetNumber: '4',
            facetName: 'Physical Equipment Facet',
            icon: Icons.devices,
            iconColor: Colors.indigo,
            metrics: [
              _MetricItem(label: 'Feeder Node', value: 'SFF-ESP32-84920 (ONLINE)'),
              _MetricItem(label: 'Hopper Feed Level', value: '78% (3.9 kg left)'),
              _MetricItem(label: 'Battery Charge', value: '94% (LiFePO4)'),
              _MetricItem(label: 'Solar Panel Voltage', value: '4.15 V (Charging)'),
              _MetricItem(label: 'Aerator Relay', value: 'ON (10 hrs/day)'),
              _MetricItem(label: 'WiFi Signal (RSSI)', value: '-58 dBm'),
            ],
          ),
          const SizedBox(height: 12),

          // FACET 5: Vision
          _FacetCard(
            facetNumber: '5',
            facetName: 'AquaVision Edge Facet',
            icon: Icons.videocam,
            iconColor: Colors.purple,
            metrics: [
              const _MetricItem(label: 'Camera Node', value: 'CAM-OV2640-3910 (ONLINE)'),
              _MetricItem(label: 'Surface Boil Score', value: '${boilScore.toInt()} / 100'),
              _MetricItem(label: 'Feeding Activity', value: boilScore > 60 ? 'HIGH (Aggressive)' : 'QUIET (Basal)'),
              _MetricItem(label: 'Gasping Score', value: isBlocked ? '84% (SURFACE PIPING)' : '4% (Normal)', isAlert: isBlocked),
              const _MetricItem(label: 'Turbidity Index', value: '14.2 NTU'),
              const _MetricItem(label: 'Model FPS', value: '12.4 fps (Edge ESP32)'),
            ],
          ),
          const SizedBox(height: 12),

          // FACET 6: Intelligence
          _FacetCard(
            facetNumber: '6',
            facetName: 'Intelligence & Decision Facet',
            icon: Icons.psychology,
            iconColor: Colors.blueAccent,
            metrics: [
              const _MetricItem(label: 'AquaDoc Agent', value: 'CONNECTED (Hybrid RAG)'),
              _MetricItem(label: 'Clinical State', value: isBlocked ? 'HYPOXIA WARNING' : 'NOMINAL (HEALTHY)', isAlert: isBlocked),
              const _MetricItem(label: 'Literature Grounding', value: 'Active (Boyd et al. 2021)'),
              const _MetricItem(label: 'Missing Parameters', value: 'Alkalinity, Hardness (UNKNOWN)'),
              const _MetricItem(label: 'Active Alerts', value: '0 Unresolved'),
              const _MetricItem(label: 'Next Recommended Action', value: 'Maintain current feeding schedule'),
            ],
          ),
          const SizedBox(height: 24),

          // Bottom Quick Action Buttons
          Row(
            children: [
              Expanded(
                child: FilledButton.icon(
                  onPressed: () => context.go('/simulator'),
                  icon: const Icon(Icons.science),
                  label: const Text('Farm Simulator'),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: () => context.go('/triage'),
                  icon: const Icon(Icons.medical_services_outlined),
                  label: const Text('Disease Triage'),
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
        ],
      ),
    );
  }
}

class _FacetCard extends StatelessWidget {
  final String facetNumber;
  final String facetName;
  final IconData icon;
  final Color iconColor;
  final List<_MetricItem> metrics;

  const _FacetCard({
    required this.facetNumber,
    required this.facetName,
    required this.icon,
    required this.iconColor,
    required this.metrics,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      elevation: 1,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Container(
                  padding: const EdgeInsets.all(6),
                  decoration: BoxDecoration(
                    color: iconColor.withOpacity(0.15),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Icon(icon, color: iconColor, size: 20),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: Text(
                    'Facet $facetNumber: $facetName',
                    style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 14),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            const Divider(height: 1),
            const SizedBox(height: 12),
            GridView.builder(
              shrinkWrap: true,
              physics: const NeverScrollableScrollPhysics(),
              gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
                crossAxisCount: 2,
                childAspectRatio: 2.8,
                crossAxisSpacing: 10,
                mainAxisSpacing: 8,
              ),
              itemCount: metrics.length,
              itemBuilder: (context, index) {
                final m = metrics[index];
                return Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Text(
                      m.label,
                      style: TextStyle(fontSize: 11, color: Colors.grey[600]),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    const SizedBox(height: 2),
                    Text(
                      m.value,
                      style: TextStyle(
                        fontSize: 13,
                        fontWeight: FontWeight.bold,
                        color: m.isAlert ? Colors.red : null,
                      ),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ],
                );
              },
            ),
          ],
        ),
      ),
    );
  }
}

class _MetricItem {
  final String label;
  final String value;
  final bool isAlert;

  const _MetricItem({
    required this.label,
    required this.value,
    this.isAlert = false,
  });
}
