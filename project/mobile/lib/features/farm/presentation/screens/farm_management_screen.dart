import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/theme/app_theme.dart';

class ProductionUnitItem {
  final String id;
  final String name;
  final String type; // pond, tank, ras, cage
  final String species;
  final int fishCount;
  final double avgWeightGrams;
  final double targetHarvestWeightGrams;
  final double dissolvedOxygen;
  final double temperature;
  final double ph;
  final double tanAmmonia;
  final String status;

  const ProductionUnitItem({
    required this.id,
    required this.name,
    required this.type,
    required this.species,
    required this.fishCount,
    required this.avgWeightGrams,
    required this.targetHarvestWeightGrams,
    required this.dissolvedOxygen,
    required this.temperature,
    required this.ph,
    required this.tanAmmonia,
    this.status = 'optimal',
  });

  double get totalBiomassKg => (fishCount * avgWeightGrams) / 1000.0;
  double get growthProgress => (avgWeightGrams / targetHarvestWeightGrams).clamp(0.0, 1.0);
}

class FarmManagementScreen extends ConsumerStatefulWidget {
  const FarmManagementScreen({super.key});

  @override
  ConsumerState<FarmManagementScreen> createState() => _FarmManagementScreenState();
}

class _FarmManagementScreenState extends ConsumerState<FarmManagementScreen> {
  final List<ProductionUnitItem> _units = [
    const ProductionUnitItem(
      id: 'unit-01',
      name: 'Earthen Pond 1 (Main Growout)',
      type: 'Earthen Pond',
      species: 'African Catfish (Clarias)',
      fishCount: 4850,
      avgWeightGrams: 320.0,
      targetHarvestWeightGrams: 800.0,
      dissolvedOxygen: 5.8,
      temperature: 28.4,
      ph: 7.4,
      tanAmmonia: 0.15,
      status: 'optimal',
    ),
    const ProductionUnitItem(
      id: 'unit-02',
      name: 'Concrete Tank 2 (Nursery)',
      type: 'Concrete Tank',
      species: 'Nile Tilapia (Oreochromis)',
      fishCount: 8200,
      avgWeightGrams: 45.0,
      targetHarvestWeightGrams: 450.0,
      dissolvedOxygen: 6.2,
      temperature: 27.9,
      ph: 7.8,
      tanAmmonia: 0.22,
      status: 'optimal',
    ),
    const ProductionUnitItem(
      id: 'unit-03',
      name: 'RAS Tank Alpha (Broodstock)',
      type: 'RAS High-Density',
      species: 'African Catfish (Broodstock)',
      fishCount: 350,
      avgWeightGrams: 1450.0,
      targetHarvestWeightGrams: 2000.0,
      dissolvedOxygen: 6.8,
      temperature: 28.1,
      ph: 7.2,
      tanAmmonia: 0.08,
      status: 'optimal',
    ),
  ];

  void _showLogSamplingDialog(ProductionUnitItem unit) {
    final weightController = TextEditingController(text: '${unit.avgWeightGrams}');
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('Sample Biometrics: ${unit.name}'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'Enter the new average fish weight (grams) measured during pond sampling.',
              style: TextStyle(fontSize: 13),
            ),
            const SizedBox(height: 16),
            TextField(
              controller: weightController,
              keyboardType: const TextInputType.numberWithOptions(decimal: true),
              decoration: const InputDecoration(
                labelText: 'Average Weight (g)',
                suffixText: 'g',
                border: OutlineInputBorder(),
              ),
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
          FilledButton(
            onPressed: () {
              final newWeight = double.tryParse(weightController.text) ?? unit.avgWeightGrams;
              setState(() {
                final idx = _units.indexWhere((u) => u.id == unit.id);
                if (idx != -1) {
                  _units[idx] = ProductionUnitItem(
                    id: unit.id,
                    name: unit.name,
                    type: unit.type,
                    species: unit.species,
                    fishCount: unit.fishCount,
                    avgWeightGrams: newWeight,
                    targetHarvestWeightGrams: unit.targetHarvestWeightGrams,
                    dissolvedOxygen: unit.dissolvedOxygen,
                    temperature: unit.temperature,
                    ph: unit.ph,
                    tanAmmonia: unit.tanAmmonia,
                  );
                }
              });
              Navigator.pop(ctx);
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(
                  content: Text('Biometrics updated for ${unit.name}! Biomass: ${((unit.fishCount * newWeight)/1000).toStringAsFixed(1)} kg'),
                  backgroundColor: AppTheme.deviceOnline,
                ),
              );
            },
            child: const Text('Save Sampling'),
          ),
        ],
      ),
    );
  }

  void _showRecordMortalityDialog(ProductionUnitItem unit) {
    final countController = TextEditingController(text: '1');
    final reasonController = TextEditingController(text: 'Natural / Non-disease');

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('Record Mortality: ${unit.name}'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: countController,
              keyboardType: TextInputType.number,
              decoration: const InputDecoration(
                labelText: 'Mortality Count (Fish)',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: reasonController,
              decoration: const InputDecoration(
                labelText: 'Observed Cause / Reason',
                border: OutlineInputBorder(),
              ),
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: AppTheme.feedLevelLow),
            onPressed: () {
              final lost = int.tryParse(countController.text) ?? 1;
              setState(() {
                final idx = _units.indexWhere((u) => u.id == unit.id);
                if (idx != -1) {
                  _units[idx] = ProductionUnitItem(
                    id: unit.id,
                    name: unit.name,
                    type: unit.type,
                    species: unit.species,
                    fishCount: (unit.fishCount - lost).clamp(0, 1000000),
                    avgWeightGrams: unit.avgWeightGrams,
                    targetHarvestWeightGrams: unit.targetHarvestWeightGrams,
                    dissolvedOxygen: unit.dissolvedOxygen,
                    temperature: unit.temperature,
                    ph: unit.ph,
                    tanAmmonia: unit.tanAmmonia,
                  );
                }
              });
              Navigator.pop(ctx);
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(
                  content: Text('Recorded $lost mortality in ${unit.name}. Cohort updated.'),
                  backgroundColor: Colors.orange[800],
                ),
              );
            },
            child: const Text('Record Event'),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final totalFarmBiomass = _units.fold<double>(0, (sum, u) => sum + u.totalBiomassKg);
    final totalFishCount = _units.fold<int>(0, (sum, u) => sum + u.fishCount);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Farm & Pond Operations'),
        actions: [
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
          // Total Farm Summary Card
          Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              gradient: LinearGradient(
                colors: [
                  Theme.of(context).colorScheme.primary,
                  Theme.of(context).colorScheme.primary.withOpacity(0.8),
                ],
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
              ),
              borderRadius: BorderRadius.circular(20),
              boxShadow: [
                BoxShadow(
                  color: Theme.of(context).colorScheme.primary.withOpacity(0.3),
                  blurRadius: 12,
                  offset: const Offset(0, 6),
                ),
              ],
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    const Text(
                      'TOTAL FARM BIOMASS',
                      style: TextStyle(
                        color: Colors.white70,
                        fontSize: 12,
                        letterSpacing: 1.2,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                      decoration: BoxDecoration(
                        color: Colors.white.withOpacity(0.2),
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Text(
                        '${_units.length} Active Units',
                        style: const TextStyle(color: Colors.white, fontSize: 12, fontWeight: FontWeight.bold),
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                Text(
                  '${totalFarmBiomass.toStringAsFixed(1)} kg',
                  style: const TextStyle(
                    color: Colors.white,
                    fontSize: 32,
                    fontWeight: FontWeight.bold,
                  ),
                ),
                const SizedBox(height: 14),
                Row(
                  children: [
                    _FarmMetricPill(
                      icon: Icons.bubble_chart,
                      label: 'Population',
                      value: '$totalFishCount fish',
                    ),
                    const SizedBox(width: 12),
                    const _FarmMetricPill(
                      icon: Icons.shield,
                      label: 'Bio-Status',
                      value: 'Nominal',
                    ),
                  ],
                ),
              ],
            ),
          ),
          const SizedBox(height: 16),

          // Digital Twin & Simulator Shortcuts Row
          Row(
            children: [
              Expanded(
                child: InkWell(
                  onTap: () => context.go('/twin'),
                  borderRadius: BorderRadius.circular(14),
                  child: Container(
                    padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
                    decoration: BoxDecoration(
                      color: const Color(0xFF0F2027),
                      borderRadius: BorderRadius.circular(14),
                      border: Border.all(color: Colors.cyan.withOpacity(0.4)),
                    ),
                    child: const Row(
                      children: [
                        Icon(Icons.hub, color: Colors.cyanAccent, size: 20),
                        SizedBox(width: 8),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text('AquaTwin 6-Facet', style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 12)),
                              Text('Live Timeline & Sensors', style: TextStyle(color: Colors.white60, fontSize: 10)),
                            ],
                          ),
                        ),
                        Icon(Icons.chevron_right, color: Colors.white38, size: 18),
                      ],
                    ),
                  ),
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: InkWell(
                  onTap: () => context.go('/simulator'),
                  borderRadius: BorderRadius.circular(14),
                  child: Container(
                    padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
                    decoration: BoxDecoration(
                      color: const Color(0xFF1A2A3A),
                      borderRadius: BorderRadius.circular(14),
                      border: Border.all(color: Colors.blueAccent.withOpacity(0.4)),
                    ),
                    child: const Row(
                      children: [
                        Icon(Icons.science, color: Colors.blueAccent, size: 20),
                        SizedBox(width: 8),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text('Farm Simulator', style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 12)),
                              Text('Growth & Policy Model', style: TextStyle(color: Colors.white60, fontSize: 10)),
                            ],
                          ),
                        ),
                        Icon(Icons.chevron_right, color: Colors.white38, size: 18),
                      ],
                    ),
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 20),

          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(
                'Production Units',
                style: Theme.of(context).textTheme.titleLarge?.copyWith(
                  fontWeight: FontWeight.bold,
                ),
              ),
              TextButton.icon(
                onPressed: () {
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('Add Unit: New pond template ready')),
                  );
                },
                icon: const Icon(Icons.add, size: 18),
                label: const Text('Add Pond'),
              ),
            ],
          ),
          const SizedBox(height: 12),

          // List of Production Units
          ..._units.map((unit) => _buildUnitCard(context, unit)),
        ],
      ),
    );
  }

  Widget _buildUnitCard(BuildContext context, ProductionUnitItem unit) {
    return Card(
      margin: const EdgeInsets.only(bottom: 16),
      elevation: 2,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Container(
                  padding: const EdgeInsets.all(8),
                  decoration: BoxDecoration(
                    color: Theme.of(context).colorScheme.primary.withOpacity(0.12),
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: Icon(
                    Icons.waves,
                    color: Theme.of(context).colorScheme.primary,
                    size: 24,
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        unit.name,
                        style: const TextStyle(
                          fontSize: 16,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                      Text(
                        '${unit.type} • ${unit.species}',
                        style: TextStyle(fontSize: 12, color: Colors.grey[600]),
                      ),
                    ],
                  ),
                ),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                  decoration: BoxDecoration(
                    color: AppTheme.deviceOnline.withOpacity(0.15),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: const Text(
                    'HEALTHY',
                    style: TextStyle(
                      color: AppTheme.deviceOnline,
                      fontSize: 10,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 16),

            // Live Telemetry Row
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: Colors.grey.withOpacity(0.06),
                borderRadius: BorderRadius.circular(12),
              ),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceAround,
                children: [
                  _TelemetryBadge(label: 'DO', value: '${unit.dissolvedOxygen} mg/L', color: Colors.blue),
                  _TelemetryBadge(label: 'Temp', value: '${unit.temperature}°C', color: Colors.orange),
                  _TelemetryBadge(label: 'pH', value: '${unit.ph}', color: Colors.green),
                  _TelemetryBadge(label: 'TAN', value: '${unit.tanAmmonia} mg/L', color: Colors.purple),
                ],
              ),
            ),
            const SizedBox(height: 16),

            // Biomass & Stocking Row
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('Total Biomass', style: TextStyle(fontSize: 12, color: Colors.grey[600])),
                    const SizedBox(height: 2),
                    Text(
                      '${unit.totalBiomassKg.toStringAsFixed(1)} kg',
                      style: const TextStyle(fontSize: 15, fontWeight: FontWeight.bold),
                    ),
                  ],
                ),
                Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('Fish Count', style: TextStyle(fontSize: 12, color: Colors.grey[600])),
                    const SizedBox(height: 2),
                    Text(
                      '${unit.fishCount}',
                      style: const TextStyle(fontSize: 15, fontWeight: FontWeight.bold),
                    ),
                  ],
                ),
                Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('Avg Weight', style: TextStyle(fontSize: 12, color: Colors.grey[600])),
                    const SizedBox(height: 2),
                    Text(
                      '${unit.avgWeightGrams.toStringAsFixed(0)} g',
                      style: const TextStyle(fontSize: 15, fontWeight: FontWeight.bold),
                    ),
                  ],
                ),
              ],
            ),
            const SizedBox(height: 12),

            // Growth Progress Bar
            ClipRRect(
              borderRadius: BorderRadius.circular(6),
              child: LinearProgressIndicator(
                value: unit.growthProgress,
                minHeight: 6,
                backgroundColor: Colors.grey.withOpacity(0.2),
                valueColor: AlwaysStoppedAnimation<Color>(Theme.of(context).colorScheme.primary),
              ),
            ),
            const SizedBox(height: 6),
            Align(
              alignment: Alignment.centerRight,
              child: Text(
                'Target: ${unit.targetHarvestWeightGrams.toInt()}g (${(unit.growthProgress * 100).toInt()}% progress)',
                style: TextStyle(fontSize: 11, color: Colors.grey[600]),
              ),
            ),
            const SizedBox(height: 12),

            // Action Buttons
            Row(
              children: [
                Expanded(
                  child: OutlinedButton.icon(
                    style: OutlinedButton.styleFrom(
                      padding: const EdgeInsets.symmetric(vertical: 8),
                    ),
                    onPressed: () => _showLogSamplingDialog(unit),
                    icon: const Icon(Icons.scale_outlined, size: 16),
                    label: const Text('Log Sampling', style: TextStyle(fontSize: 12)),
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: OutlinedButton.icon(
                    style: OutlinedButton.styleFrom(
                      foregroundColor: Colors.orange[800],
                      padding: const EdgeInsets.symmetric(vertical: 8),
                    ),
                    onPressed: () => _showRecordMortalityDialog(unit),
                    icon: const Icon(Icons.warning_amber_outlined, size: 16),
                    label: const Text('Mortality', style: TextStyle(fontSize: 12)),
                  ),
                ),
                const SizedBox(width: 8),
                IconButton(
                  style: IconButton.styleFrom(
                    backgroundColor: Theme.of(context).colorScheme.primary.withOpacity(0.1),
                  ),
                  tooltip: 'AquaDoc Clinical Check',
                  onPressed: () => context.go('/aquadoc'),
                  icon: Icon(Icons.psychology, color: Theme.of(context).colorScheme.primary, size: 20),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _FarmMetricPill extends StatelessWidget {
  final IconData icon;
  final String label;
  final String value;

  const _FarmMetricPill({
    required this.icon,
    required this.label,
    required this.value,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: Colors.white.withOpacity(0.18),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, color: Colors.white, size: 14),
          const SizedBox(width: 6),
          Text(
            '$label: $value',
            style: const TextStyle(color: Colors.white, fontSize: 11, fontWeight: FontWeight.w600),
          ),
        ],
      ),
    );
  }
}

class _TelemetryBadge extends StatelessWidget {
  final String label;
  final String value;
  final Color color;

  const _TelemetryBadge({
    required this.label,
    required this.value,
    required this.color,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Text(
          label,
          style: TextStyle(fontSize: 11, color: Colors.grey[600], fontWeight: FontWeight.w500),
        ),
        const SizedBox(height: 2),
        Text(
          value,
          style: TextStyle(fontSize: 13, color: color, fontWeight: FontWeight.bold),
        ),
      ],
    );
  }
}
