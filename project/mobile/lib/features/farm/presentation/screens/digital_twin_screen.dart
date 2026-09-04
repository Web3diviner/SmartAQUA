import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/models/device.dart';
import '../../../../core/models/farm_unit.dart';
import '../../../../core/providers/device_provider.dart';
import '../../../../core/providers/farm_provider.dart';
import '../../../../core/providers/monitoring_provider.dart';
import '../widgets/digital_twin_3d_visualizer.dart';

class DigitalTwinScreen extends ConsumerStatefulWidget {
  const DigitalTwinScreen({super.key});

  @override
  ConsumerState<DigitalTwinScreen> createState() => _DigitalTwinScreenState();
}

class _DigitalTwinScreenState extends ConsumerState<DigitalTwinScreen> {
  double _timelineHour = 14.0; // 14:00 (2:00 PM)
  String? _selectedUnitId;

  @override
  Widget build(BuildContext context) {
    final farmUnits = ref.watch(farmUnitsProvider).units;
    final devices = ref.watch(devicesProvider);
    final sensorData = ref.watch(sensorDataProvider).currentData;

    // Resolve selected farm unit
    FarmUnit? activeUnit;
    if (farmUnits.isNotEmpty) {
      if (_selectedUnitId != null && farmUnits.any((u) => u.id == _selectedUnitId)) {
        activeUnit = farmUnits.firstWhere((u) => u.id == _selectedUnitId);
      } else {
        activeUnit = farmUnits.first;
        _selectedUnitId = activeUnit.id;
      }
    }

    // Resolve Physical Dimensions & Fish Stock
    final tankLength = activeUnit?.lengthM ?? 15.0;
    final tankWidth = activeUnit?.widthM ?? 9.0;
    final tankDepth = activeUnit?.depthM ?? 1.5;
    final population = activeUnit?.fishCount ?? 3000;
    final avgWeight = activeUnit?.avgWeightGrams ?? 250.0;
    final biomass = activeUnit?.totalBiomassKg ?? (population * avgWeight / 1000.0);
    final species = activeUnit?.species ?? 'African Catfish (Clarias gariepinus)';
    final productionDays = activeUnit?.daysInProduction ?? 60;
    final volumeM3 = activeUnit?.volumeM3 ?? (tankLength * tankWidth * tankDepth);
    final density = volumeM3 > 0 ? (biomass / volumeM3) : 0.0;

    // Resolve Environmental Telemetry: Hardware Sensor > Manual Lab Entry > Timeline Estimation
    final liveTemp = (sensorData != null && sensorData.waterTemperature > 0) ? sensorData.waterTemperature : null;
    final manualTemp = activeUnit?.manualTemp;
    final baseTemp = liveTemp ?? manualTemp ?? 28.0;
    final tempValue = (baseTemp + 0.8 * (1 - ((_timelineHour - 14).abs() / 12))).clamp(22.0, 34.0);

    final manualDO = activeUnit?.manualDO;
    final baseDO = manualDO ?? 5.5;
    final doValue = (baseDO + 0.9 * (1 - ((_timelineHour - 14).abs() / 12))).clamp(2.4, 8.0);

    final manualTAN = activeUnit?.manualTAN;
    final baseTAN = manualTAN ?? 0.18;
    final tanValue = (baseTAN + (_timelineHour >= 12 && _timelineHour <= 18 ? 0.04 : 0.0)).clamp(0.02, 1.2);

    final boilScore = (_timelineHour == 8 || _timelineHour == 13 || _timelineHour == 17) ? 82.0 : 24.0;
    final isBlocked = doValue < 3.0;

    // Estimated Survival & Mortality
    final estMortality = ((productionDays * 0.0008 + (doValue < 3.5 ? 0.03 : 0.0)) * population).clamp(0, population * 0.4).toInt();
    final survivalRate = population > 0 ? (((population - estMortality) / population) * 100.0).clamp(60.0, 100.0) : 100.0;

    // Linked hardware device for selected unit
    Device? linkedDevice;
    if (activeUnit?.linkedDeviceId != null) {
      linkedDevice = devices.where((d) => d.id == activeUnit!.linkedDeviceId).firstOrNull;
    }

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
          // Unit Selector & State Header
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
                    if (farmUnits.isNotEmpty)
                      DropdownButtonHideUnderline(
                        child: DropdownButton<String>(
                          value: _selectedUnitId,
                          dropdownColor: const Color(0xFF203A43),
                          style: const TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 16),
                          icon: const Icon(Icons.arrow_drop_down, color: Colors.white),
                          items: farmUnits
                              .map((u) => DropdownMenuItem(
                                    value: u.id,
                                    child: Text('${u.name} (${u.volumeM3.toStringAsFixed(0)} m³)'),
                                  ))
                              .toList(),
                          onChanged: (val) {
                            if (val != null) setState(() => _selectedUnitId = val);
                          },
                        ),
                      )
                    else
                      const Text(
                        'Sandbox / Unconfigured Farm',
                        style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 16),
                      ),
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                      decoration: BoxDecoration(
                        color: isBlocked ? Colors.red : Colors.green.withOpacity(0.25),
                        borderRadius: BorderRadius.circular(12),
                        border: Border.all(color: isBlocked ? Colors.white : Colors.greenAccent),
                      ),
                      child: Text(
                        isBlocked ? 'SAFETY INTERLOCK ACTIVE' : 'TWIN STATE: NOMINAL',
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
                      'Biomass: ${biomass.toStringAsFixed(1)} kg',
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
          const SizedBox(height: 16),

          if (farmUnits.isEmpty)
            Card(
              color: Colors.blueGrey.withOpacity(0.15),
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Row(
                  children: [
                    const Icon(Icons.info_outline, color: Colors.cyanAccent),
                    const SizedBox(width: 12),
                    const Expanded(
                      child: Text(
                        'You are viewing a Sandbox simulation. Add your real ponds in Farm Operations to synchronize physical dimensions and stock counts.',
                        style: TextStyle(fontSize: 12),
                      ),
                    ),
                    const SizedBox(width: 8),
                    FilledButton(
                      style: FilledButton.styleFrom(padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6)),
                      onPressed: () => context.go('/farm'),
                      child: const Text('Add Pond', style: TextStyle(fontSize: 11)),
                    ),
                  ],
                ),
              ),
            ),
          if (farmUnits.isEmpty) const SizedBox(height: 16),

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
            avgWeightG: avgWeight,
            biomassKg: biomass,
            population: population,
            tankLengthM: tankLength,
            tankWidthM: tankWidth,
            tankDepthM: tankDepth,
            productionPeriodDays: productionDays,
            survivalRate: survivalRate,
            mortalityCount: estMortality,
            species: species,
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
              _MetricItem(
                label: 'Dissolved Oxygen',
                value: manualDO != null ? '${doValue.toStringAsFixed(1)} mg/L (Lab)' : '${doValue.toStringAsFixed(1)} mg/L (Sim)',
                isAlert: doValue < 3.0,
              ),
              _MetricItem(
                label: 'Water Temperature',
                value: liveTemp != null ? '${tempValue.toStringAsFixed(1)} °C (Live)' : '${tempValue.toStringAsFixed(1)} °C',
                isAlert: tempValue < 20 || tempValue > 34,
              ),
              _MetricItem(
                label: 'pH Level',
                value: activeUnit?.manualPh != null ? activeUnit!.manualPh!.toStringAsFixed(1) : '7.2 (Nominal)',
                isAlert: false,
              ),
              _MetricItem(
                label: 'TAN Ammonia',
                value: manualTAN != null ? '${tanValue.toStringAsFixed(2)} mg/L (Lab)' : '${tanValue.toStringAsFixed(2)} mg/L (Sim)',
                isAlert: tanValue > 0.8,
              ),
              _MetricItem(label: 'DO Saturation', value: '${(doValue / 7.8 * 100).toInt()}%', isAlert: false),
              _MetricItem(label: 'Nitrite (NO2)', value: '0.02 mg/L', isAlert: false),
            ],
          ),
          const SizedBox(height: 12),

          // FACET 2: Biological & Fish Stock
          _FacetCard(
            facetNumber: '2',
            facetName: 'Biological & Fish Facet',
            icon: Icons.set_meal,
            iconColor: Colors.teal,
            metrics: [
              _MetricItem(label: 'Species', value: species.split('(').first.trim()),
              _MetricItem(label: 'Initial Stock Set', value: '$population fish'),
              _MetricItem(label: 'Surviving Stock', value: '${population - estMortality} fish (${survivalRate.toStringAsFixed(1)}%)'),
              _MetricItem(label: 'Cumulative Mortality', value: '$estMortality fish'),
              _MetricItem(label: 'Stocking Density', value: '${density.toStringAsFixed(1)} kg/m³ (${volumeM3.toStringAsFixed(0)} m³)'),
              _MetricItem(label: 'Production Timeline', value: 'Day $productionDays / 180'),
              _MetricItem(label: 'Average Weight', value: '${avgWeight.toStringAsFixed(0)} g'),
              _MetricItem(label: 'Total Standing Biomass', value: '${biomass.toStringAsFixed(1)} kg'),
            ],
          ),
          const SizedBox(height: 12),

          // FACET 3: Feeding Automation
          _FacetCard(
            facetNumber: '3',
            facetName: 'Feeding Automation Facet',
            icon: Icons.restaurant,
            iconColor: Colors.orange,
            metrics: [
              _MetricItem(label: 'Daily Target Feed', value: '${(biomass * 0.025).toStringAsFixed(1)} kg (2.5% BW)'),
              _MetricItem(label: 'Target FCR', value: '1.20'),
              _MetricItem(label: 'Q10 Factor', value: '${(1.0 + (tempValue - 28.0) * 0.05).clamp(0.8, 1.4).toStringAsFixed(2)}x'),
              _MetricItem(label: 'Interlock Status', value: isBlocked ? 'LOCKED (NO FEED)' : 'UNLOCKED (SAFE)', isAlert: isBlocked),
              _MetricItem(label: 'Rations / Day', value: '3 feeds (08:00, 13:00, 17:00)'),
              _MetricItem(label: 'Next Feed Quantity', value: '${(biomass * 0.025 / 3.0).toStringAsFixed(2)} kg'),
            ],
          ),
          const SizedBox(height: 12),

          // FACET 4: Equipment & Actuation
          _FacetCard(
            facetNumber: '4',
            facetName: 'Physical Equipment Facet',
            icon: Icons.devices,
            iconColor: Colors.indigo,
            metrics: [
              _MetricItem(
                label: 'Linked Feeder Node',
                value: linkedDevice != null ? '${linkedDevice.name} (${linkedDevice.isOnline ? "ONLINE" : "OFFLINE"})' : 'No Hardware Node',
              ),
              _MetricItem(
                label: 'Hopper Feed Level',
                value: (sensorData != null && sensorData.feedLevel > 0) ? '${sensorData.feedLevel.toInt()}% Full' : (linkedDevice != null ? 'Connected' : 'Unlinked'),
              ),
              _MetricItem(
                label: 'Battery Charge',
                value: (sensorData != null && sensorData.batteryLevel > 0) ? '${sensorData.batteryLevel.toInt()}%' : '--',
              ),
              _MetricItem(
                label: 'Solar Voltage',
                value: (sensorData != null && sensorData.solarVoltage > 0) ? '${sensorData.solarVoltage.toStringAsFixed(1)} V' : '--',
              ),
            ],
          ),
          const SizedBox(height: 12),

          // FACET 5: Vision & Behavior
          _FacetCard(
            facetNumber: '5',
            facetName: 'AquaVision Edge Facet',
            icon: Icons.videocam,
            iconColor: Colors.purple,
            metrics: [
              _MetricItem(label: 'Surface Boil Score', value: '${boilScore.toInt()} / 100'),
              _MetricItem(label: 'Feeding Appetite', value: boilScore > 60 ? 'HIGH (Aggressive)' : 'NORMAL'),
              _MetricItem(label: 'Gasping Indicator', value: isBlocked ? 'SURFACE PIPING (DO Low)' : 'Normal', isAlert: isBlocked),
              const _MetricItem(label: 'Turbidity Index', value: 'Normal'),
            ],
          ),
          const SizedBox(height: 12),

          // FACET 6: Intelligence & Bioenergetics
          _FacetCard(
            facetNumber: '6',
            facetName: 'Intelligence & Decision Facet',
            icon: Icons.psychology,
            iconColor: Colors.blueAccent,
            metrics: [
              const _MetricItem(label: 'AquaDoc Agent', value: 'ACTIVE (Grounding)'),
              _MetricItem(label: 'Clinical Assessment', value: isBlocked ? 'HYPOXIA INTERVENT' : 'NOMINAL HEALTH', isAlert: isBlocked),
              _MetricItem(label: 'Harvest Projection', value: '${activeUnit?.targetHarvestWeightGrams.toInt() ?? 800}g Target'),
              _MetricItem(label: 'Biosecurity Risk', value: doValue < 3.5 ? 'ELEVATED' : 'LOW RISK', isAlert: doValue < 3.5),
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
