import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/models/farm_unit.dart';
import '../../../../core/providers/device_provider.dart';
import '../../../../core/providers/farm_provider.dart';
import '../../../../core/providers/monitoring_provider.dart';
import '../../../../core/theme/app_theme.dart';

class FarmManagementScreen extends ConsumerStatefulWidget {
  const FarmManagementScreen({super.key});

  @override
  ConsumerState<FarmManagementScreen> createState() => _FarmManagementScreenState();
}

class _FarmManagementScreenState extends ConsumerState<FarmManagementScreen> {
  @override
  void initState() {
    super.initState();
    Future.microtask(() {
      ref.read(deviceListProvider.notifier).loadDevices();
    });
  }

  void _showAddUnitDialog() {
    final nameController = TextEditingController(text: 'Pond ${_unitsCount() + 1}');
    String selectedType = 'Earthen Pond';
    String selectedSpecies = 'African Catfish (Clarias gariepinus)';
    final populationController = TextEditingController(text: '3000');
    final weightController = TextEditingController(text: '150');
    final targetWeightController = TextEditingController(text: '800');
    final lengthController = TextEditingController(text: '15.0');
    final widthController = TextEditingController(text: '9.0');
    final depthController = TextEditingController(text: '1.5');
    String? selectedDeviceId;

    final devices = ref.read(devicesProvider);

    showDialog(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setDialogState) => AlertDialog(
          title: const Row(
            children: [
              Icon(Icons.add_circle_outline, color: AppTheme.primaryCyan),
              SizedBox(width: 8),
              Text('Add Production Unit'),
            ],
          ),
          content: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                TextField(
                  controller: nameController,
                  decoration: const InputDecoration(labelText: 'Unit Name / Label'),
                ),
                const SizedBox(height: 12),
                DropdownButtonFormField<String>(
                  value: selectedType,
                  decoration: const InputDecoration(labelText: 'System / Enclosure Type'),
                  items: const [
                    DropdownMenuItem(value: 'Earthen Pond', child: Text('Earthen Pond')),
                    DropdownMenuItem(value: 'Concrete Tank', child: Text('Concrete Tank')),
                    DropdownMenuItem(value: 'Tarpaulin Tank', child: Text('Tarpaulin Tank')),
                    DropdownMenuItem(value: 'RAS High-Density', child: Text('RAS High-Density')),
                    DropdownMenuItem(value: 'Cage Enclosure', child: Text('Cage Enclosure')),
                  ],
                  onChanged: (val) {
                    if (val != null) setDialogState(() => selectedType = val);
                  },
                ),
                const SizedBox(height: 12),
                DropdownButtonFormField<String>(
                  value: selectedSpecies,
                  decoration: const InputDecoration(labelText: 'Cultured Species'),
                  items: const [
                    DropdownMenuItem(value: 'African Catfish (Clarias gariepinus)', child: Text('African Catfish (Clarias)')),
                    DropdownMenuItem(value: 'Nile Tilapia (Oreochromis niloticus)', child: Text('Nile Tilapia (Oreochromis)')),
                    DropdownMenuItem(value: 'Heterobranchus longifilis', child: Text('Heterobranchus (Vundu)')),
                    DropdownMenuItem(value: 'Heterobranchus × Clarias Hybrid', child: Text('Hetero-clarias Hybrid')),
                  ],
                  onChanged: (val) {
                    if (val != null) setDialogState(() => selectedSpecies = val);
                  },
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    Expanded(
                      child: TextField(
                        controller: populationController,
                        keyboardType: TextInputType.number,
                        decoration: const InputDecoration(labelText: 'Initial Stock Count', suffixText: 'fish'),
                      ),
                    ),
                    const SizedBox(width: 10),
                    Expanded(
                      child: TextField(
                        controller: weightController,
                        keyboardType: const TextInputType.numberWithOptions(decimal: true),
                        decoration: const InputDecoration(labelText: 'Current Avg Weight', suffixText: 'g'),
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: targetWeightController,
                  keyboardType: const TextInputType.numberWithOptions(decimal: true),
                  decoration: const InputDecoration(labelText: 'Target Harvest Weight', suffixText: 'g'),
                ),
                const SizedBox(height: 16),
                const Text(
                  'Physical Dimensions (Meters)',
                  style: TextStyle(fontWeight: FontWeight.bold, fontSize: 12, color: Colors.cyanAccent),
                ),
                const SizedBox(height: 8),
                Row(
                  children: [
                    Expanded(
                      child: TextField(
                        controller: lengthController,
                        keyboardType: const TextInputType.numberWithOptions(decimal: true),
                        decoration: const InputDecoration(labelText: 'Length (m)'),
                      ),
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: TextField(
                        controller: widthController,
                        keyboardType: const TextInputType.numberWithOptions(decimal: true),
                        decoration: const InputDecoration(labelText: 'Width (m)'),
                      ),
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: TextField(
                        controller: depthController,
                        keyboardType: const TextInputType.numberWithOptions(decimal: true),
                        decoration: const InputDecoration(labelText: 'Depth (m)'),
                      ),
                    ),
                  ],
                ),
                if (devices.isNotEmpty) ...[
                  const SizedBox(height: 16),
                  DropdownButtonFormField<String?>(
                    value: selectedDeviceId,
                    decoration: const InputDecoration(labelText: 'Link Hardware Feeder Node (Optional)'),
                    items: [
                      const DropdownMenuItem(value: null, child: Text('No Hardware Link (Manual unit)')),
                      ...devices.map((d) => DropdownMenuItem(
                            value: d.id,
                            child: Text('${d.name} (${d.isOnline ? "Online" : "Offline"})'),
                          )),
                    ],
                    onChanged: (val) => setDialogState(() => selectedDeviceId = val),
                  ),
                ],
              ],
            ),
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
            FilledButton(
              onPressed: () {
                final pop = int.tryParse(populationController.text) ?? 1000;
                final weight = double.tryParse(weightController.text) ?? 100.0;
                final targetW = double.tryParse(targetWeightController.text) ?? 800.0;
                final length = double.tryParse(lengthController.text) ?? 12.0;
                final width = double.tryParse(widthController.text) ?? 8.0;
                final depth = double.tryParse(depthController.text) ?? 1.5;

                final newUnit = FarmUnit(
                  id: 'unit-${DateTime.now().millisecondsSinceEpoch}',
                  name: nameController.text.trim().isEmpty ? 'Pond' : nameController.text.trim(),
                  type: selectedType,
                  species: selectedSpecies,
                  fishCount: pop,
                  avgWeightGrams: weight,
                  targetHarvestWeightGrams: targetW,
                  lengthM: length,
                  widthM: width,
                  depthM: depth,
                  linkedDeviceId: selectedDeviceId,
                  stockedAt: DateTime.now(),
                  lastSampledAt: DateTime.now(),
                );

                ref.read(farmUnitsProvider.notifier).addUnit(newUnit);
                Navigator.pop(ctx);
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(
                    content: Text('Added ${newUnit.name}! Dimensions: ${length}m × ${width}m (${newUnit.volumeM3.toStringAsFixed(0)} m³)'),
                    backgroundColor: AppTheme.deviceOnline,
                  ),
                );
              },
              child: const Text('Create Unit'),
            ),
          ],
        ),
      ),
    );
  }

  int _unitsCount() {
    return ref.read(farmUnitsProvider).units.length;
  }

  void _showEditUnitDialog(FarmUnit unit) {
    final nameController = TextEditingController(text: unit.name);
    final countController = TextEditingController(text: '${unit.fishCount}');
    final targetController = TextEditingController(text: '${unit.targetHarvestWeightGrams.toInt()}');
    final lengthController = TextEditingController(text: '${unit.lengthM}');
    final widthController = TextEditingController(text: '${unit.widthM}');
    final depthController = TextEditingController(text: '${unit.depthM}');

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('Edit ${unit.name}'),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: nameController,
                decoration: const InputDecoration(labelText: 'Unit Name'),
              ),
              const SizedBox(height: 10),
              TextField(
                controller: countController,
                keyboardType: TextInputType.number,
                decoration: const InputDecoration(labelText: 'Current Fish Stock Count', suffixText: 'fish'),
              ),
              const SizedBox(height: 10),
              TextField(
                controller: targetController,
                keyboardType: const TextInputType.numberWithOptions(decimal: true),
                decoration: const InputDecoration(labelText: 'Target Harvest Weight', suffixText: 'g'),
              ),
              const SizedBox(height: 14),
              const Align(
                alignment: Alignment.centerLeft,
                child: Text('Dimensions (m)', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 12)),
              ),
              const SizedBox(height: 6),
              Row(
                children: [
                  Expanded(
                    child: TextField(
                      controller: lengthController,
                      keyboardType: const TextInputType.numberWithOptions(decimal: true),
                      decoration: const InputDecoration(labelText: 'Length'),
                    ),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: TextField(
                      controller: widthController,
                      keyboardType: const TextInputType.numberWithOptions(decimal: true),
                      decoration: const InputDecoration(labelText: 'Width'),
                    ),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: TextField(
                      controller: depthController,
                      keyboardType: const TextInputType.numberWithOptions(decimal: true),
                      decoration: const InputDecoration(labelText: 'Depth'),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
          FilledButton(
            onPressed: () {
              final updated = unit.copyWith(
                name: nameController.text.trim().isEmpty ? unit.name : nameController.text.trim(),
                fishCount: int.tryParse(countController.text) ?? unit.fishCount,
                targetHarvestWeightGrams: double.tryParse(targetController.text) ?? unit.targetHarvestWeightGrams,
                lengthM: double.tryParse(lengthController.text) ?? unit.lengthM,
                widthM: double.tryParse(widthController.text) ?? unit.widthM,
                depthM: double.tryParse(depthController.text) ?? unit.depthM,
              );
              ref.read(farmUnitsProvider.notifier).updateUnit(updated);
              Navigator.pop(ctx);
            },
            child: const Text('Save Changes'),
          ),
        ],
      ),
    );
  }

  void _showLogSamplingDialog(FarmUnit unit) {
    final weightController = TextEditingController(text: '${unit.avgWeightGrams.toInt()}');
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('Sample Biometrics: ${unit.name}'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'Enter the measured average fish weight from sampling:',
              style: TextStyle(fontSize: 13),
            ),
            const SizedBox(height: 16),
            TextField(
              controller: weightController,
              keyboardType: const TextInputType.numberWithOptions(decimal: true),
              decoration: const InputDecoration(
                labelText: 'Average Fish Weight (g)',
                suffixText: 'g',
              ),
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
          FilledButton(
            onPressed: () {
              final newWeight = double.tryParse(weightController.text) ?? unit.avgWeightGrams;
              ref.read(farmUnitsProvider.notifier).recordSampling(unit.id, newWeight);
              Navigator.pop(ctx);
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(
                  content: Text('Sampling saved for ${unit.name}! Biomass: ${((unit.fishCount * newWeight) / 1000).toStringAsFixed(1)} kg'),
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

  void _showRecordMortalityDialog(FarmUnit unit) {
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
              ),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: reasonController,
              decoration: const InputDecoration(
                labelText: 'Observed Cause / Reason',
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
              ref.read(farmUnitsProvider.notifier).recordMortality(unit.id, lost);
              Navigator.pop(ctx);
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(
                  content: Text('Recorded $lost mortality in ${unit.name}. Stock updated.'),
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

  void _showManualWaterQualityDialog(FarmUnit unit) {
    final doController = TextEditingController(text: unit.manualDO != null ? '${unit.manualDO}' : '');
    final tempController = TextEditingController(text: unit.manualTemp != null ? '${unit.manualTemp}' : '');
    final phController = TextEditingController(text: unit.manualPh != null ? '${unit.manualPh}' : '');
    final tanController = TextEditingController(text: unit.manualTAN != null ? '${unit.manualTAN}' : '');

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('Water Quality Entry: ${unit.name}'),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Text(
                'Enter manual test-kit measurements or calibrate probe readings:',
                style: TextStyle(fontSize: 12, color: Colors.grey),
              ),
              const SizedBox(height: 14),
              TextField(
                controller: doController,
                keyboardType: const TextInputType.numberWithOptions(decimal: true),
                decoration: const InputDecoration(labelText: 'Dissolved Oxygen (DO)', suffixText: 'mg/L'),
              ),
              const SizedBox(height: 10),
              TextField(
                controller: tempController,
                keyboardType: const TextInputType.numberWithOptions(decimal: true),
                decoration: const InputDecoration(labelText: 'Water Temperature', suffixText: '°C'),
              ),
              const SizedBox(height: 10),
              TextField(
                controller: phController,
                keyboardType: const TextInputType.numberWithOptions(decimal: true),
                decoration: const InputDecoration(labelText: 'pH Level'),
              ),
              const SizedBox(height: 10),
              TextField(
                controller: tanController,
                keyboardType: const TextInputType.numberWithOptions(decimal: true),
                decoration: const InputDecoration(labelText: 'Total Ammonia Nitrogen (TAN)', suffixText: 'mg/L'),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
          FilledButton(
            onPressed: () {
              ref.read(farmUnitsProvider.notifier).updateManualWaterQuality(
                    unit.id,
                    doMgL: double.tryParse(doController.text),
                    tempC: double.tryParse(tempController.text),
                    ph: double.tryParse(phController.text),
                    tanMgL: double.tryParse(tanController.text),
                  );
              Navigator.pop(ctx);
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(
                  content: Text('Water quality parameters updated!'),
                  backgroundColor: AppTheme.deviceOnline,
                ),
              );
            },
            child: const Text('Save Readings'),
          ),
        ],
      ),
    );
  }

  void _confirmDeleteUnit(FarmUnit unit) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('Delete ${unit.name}?'),
        content: const Text('Are you sure you want to delete this pond unit? This will permanently remove its biometric logs and dimensions.'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Colors.red),
            onPressed: () {
              ref.read(farmUnitsProvider.notifier).deleteUnit(unit.id);
              Navigator.pop(ctx);
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(content: Text('Deleted ${unit.name}.')),
              );
            },
            child: const Text('Delete'),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final farmState = ref.watch(farmUnitsProvider);
    final units = farmState.units;

    final totalFarmBiomass = units.fold<double>(0, (sum, u) => sum + u.totalBiomassKg);
    final totalFishCount = units.fold<int>(0, (sum, u) => sum + u.fishCount);

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
                  const Color(0xFF0F2027),
                ],
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
              ),
              borderRadius: BorderRadius.circular(20),
              boxShadow: [
                BoxShadow(
                  color: Theme.of(context).colorScheme.primary.withValues(alpha: 0.25),
                  blurRadius: 14,
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
                      'TOTAL STANDING BIOMASS',
                      style: TextStyle(
                        color: Colors.white70,
                        fontSize: 11,
                        letterSpacing: 1.2,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                      decoration: BoxDecoration(
                        color: Colors.white.withValues(alpha: 0.2),
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Text(
                        '${units.length} Active Unit${units.length == 1 ? '' : 's'}',
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
                      label: 'Total Stock',
                      value: '$totalFishCount fish',
                    ),
                    const SizedBox(width: 12),
                    _FarmMetricPill(
                      icon: Icons.aspect_ratio,
                      label: 'Water Volume',
                      value: '${units.fold<double>(0, (s, u) => s + u.volumeM3).toStringAsFixed(0)} m³',
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
                      border: Border.all(color: Colors.cyan.withValues(alpha: 0.4)),
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
                              Text('Live 3D & Timeline', style: TextStyle(color: Colors.white60, fontSize: 10)),
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
                      border: Border.all(color: Colors.blueAccent.withValues(alpha: 0.4)),
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
                              Text('Mortality & Growth', style: TextStyle(color: Colors.white60, fontSize: 10)),
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
              FilledButton.icon(
                onPressed: _showAddUnitDialog,
                icon: const Icon(Icons.add, size: 18),
                label: const Text('Add Unit / Pond'),
              ),
            ],
          ),
          const SizedBox(height: 12),

          if (units.isEmpty)
            Card(
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
              child: Padding(
                padding: const EdgeInsets.all(32),
                child: Column(
                  children: [
                    const Icon(Icons.waves, size: 48, color: Colors.grey),
                    const SizedBox(height: 12),
                    const Text('No Production Units Configured', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
                    const SizedBox(height: 6),
                    const Text(
                      'Add your ponds or tanks to configure their dimensions, stocking population, and track live sensor telemetry.',
                      textAlign: TextAlign.center,
                      style: TextStyle(color: Colors.grey, fontSize: 12),
                    ),
                    const SizedBox(height: 16),
                    FilledButton.icon(
                      onPressed: _showAddUnitDialog,
                      icon: const Icon(Icons.add),
                      label: const Text('Add First Unit'),
                    ),
                  ],
                ),
              ),
            ),

          // List of Real User-Defined Production Units
          ...units.map((unit) => _buildUnitCard(context, unit)),
        ],
      ),
    );
  }

  Widget _buildUnitCard(BuildContext context, FarmUnit unit) {
    final sensorData = ref.watch(sensorDataProvider).currentData;
    final hasLiveDevice = unit.linkedDeviceId != null && sensorData != null;

    // Resolve Telemetry: Hardware Sensor > Manual Lab Entry > Unmeasured
    final tempDisplay = hasLiveDevice && sensorData.waterTemperature > 0
        ? '${sensorData.waterTemperature.toStringAsFixed(1)}°C'
        : (unit.manualTemp != null ? '${unit.manualTemp!.toStringAsFixed(1)}°C (manual)' : '-- (Unmeasured)');

    final doDisplay = unit.manualDO != null
        ? '${unit.manualDO!.toStringAsFixed(1)} mg/L'
        : '-- mg/L (No Probe)';

    final phDisplay = unit.manualPh != null
        ? unit.manualPh!.toStringAsFixed(1)
        : '-- (No Probe)';

    final tanDisplay = unit.manualTAN != null
        ? '${unit.manualTAN!.toStringAsFixed(2)} mg/L'
        : '-- mg/L (No Probe)';

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
                    color: Theme.of(context).colorScheme.primary.withValues(alpha: 0.12),
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
                        '${unit.type} • ${unit.lengthM.toStringAsFixed(1)}m × ${unit.widthM.toStringAsFixed(1)}m × ${unit.depthM.toStringAsFixed(1)}m (${unit.volumeM3.toStringAsFixed(0)} m³)',
                        style: TextStyle(fontSize: 11.5, color: Colors.grey[400]),
                      ),
                    ],
                  ),
                ),
                PopupMenuButton<String>(
                  icon: const Icon(Icons.more_vert, size: 20),
                  onSelected: (val) {
                    if (val == 'edit') _showEditUnitDialog(unit);
                    if (val == 'water') _showManualWaterQualityDialog(unit);
                    if (val == 'delete') _confirmDeleteUnit(unit);
                  },
                  itemBuilder: (ctx) => [
                    const PopupMenuItem(value: 'edit', child: Row(children: [Icon(Icons.edit, size: 16), SizedBox(width: 8), Text('Edit Unit & Dimensions')])),
                    const PopupMenuItem(value: 'water', child: Row(children: [Icon(Icons.science, size: 16), SizedBox(width: 8), Text('Enter Water Measurements')])),
                    const PopupMenuItem(value: 'delete', child: Row(children: [Icon(Icons.delete, color: Colors.red, size: 16), SizedBox(width: 8), Text('Delete Unit', style: TextStyle(color: Colors.red))])),
                  ],
                ),
              ],
            ),
            const SizedBox(height: 12),

            // Live Telemetry or Unmeasured Badges
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: Colors.black.withValues(alpha: 0.2),
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: Colors.white10),
              ),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceAround,
                children: [
                  _TelemetryBadge(label: 'DO', value: doDisplay, color: unit.manualDO != null ? Colors.blue : Colors.grey),
                  _TelemetryBadge(label: 'Temp', value: tempDisplay, color: (hasLiveDevice || unit.manualTemp != null) ? Colors.orange : Colors.grey),
                  _TelemetryBadge(label: 'pH', value: phDisplay, color: unit.manualPh != null ? Colors.green : Colors.grey),
                  _TelemetryBadge(label: 'TAN', value: tanDisplay, color: unit.manualTAN != null ? Colors.purple : Colors.grey),
                ],
              ),
            ),
            const SizedBox(height: 14),

            // Biomass & Stocking Row
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('Total Biomass', style: TextStyle(fontSize: 11, color: Colors.grey[400])),
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
                    Text('Stock Count', style: TextStyle(fontSize: 11, color: Colors.grey[400])),
                    const SizedBox(height: 2),
                    Text(
                      '${unit.fishCount} fish',
                      style: const TextStyle(fontSize: 15, fontWeight: FontWeight.bold),
                    ),
                  ],
                ),
                Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('Stocking Density', style: TextStyle(fontSize: 11, color: Colors.grey[400])),
                    const SizedBox(height: 2),
                    Text(
                      '${unit.densityKgM3.toStringAsFixed(1)} kg/m³',
                      style: const TextStyle(fontSize: 15, fontWeight: FontWeight.bold, color: Colors.cyanAccent),
                    ),
                  ],
                ),
                Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('Avg Weight', style: TextStyle(fontSize: 11, color: Colors.grey[400])),
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
                backgroundColor: Colors.grey.withValues(alpha: 0.2),
                valueColor: AlwaysStoppedAnimation<Color>(Theme.of(context).colorScheme.primary),
              ),
            ),
            const SizedBox(height: 4),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(
                  'Day ${unit.daysInProduction} in Production (${unit.species.split(" ").first})',
                  style: TextStyle(fontSize: 10.5, color: Colors.grey[400]),
                ),
                Text(
                  'Target: ${unit.targetHarvestWeightGrams.toInt()}g (${(unit.growthProgress * 100).toInt()}%)',
                  style: TextStyle(fontSize: 10.5, color: Colors.grey[400]),
                ),
              ],
            ),
            const SizedBox(height: 12),

            // Action Buttons
            Row(
              children: [
                Expanded(
                  child: OutlinedButton.icon(
                    style: OutlinedButton.styleFrom(padding: const EdgeInsets.symmetric(vertical: 8)),
                    onPressed: () => _showLogSamplingDialog(unit),
                    icon: const Icon(Icons.scale_outlined, size: 15),
                    label: const Text('Log Sampling', style: TextStyle(fontSize: 11.5)),
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: OutlinedButton.icon(
                    style: OutlinedButton.styleFrom(
                      foregroundColor: Colors.orangeAccent,
                      padding: const EdgeInsets.symmetric(vertical: 8),
                    ),
                    onPressed: () => _showRecordMortalityDialog(unit),
                    icon: const Icon(Icons.warning_amber_outlined, size: 15),
                    label: const Text('Mortality', style: TextStyle(fontSize: 11.5)),
                  ),
                ),
                const SizedBox(width: 8),
                IconButton(
                  style: IconButton.styleFrom(
                    backgroundColor: Theme.of(context).colorScheme.primary.withValues(alpha: 0.15),
                  ),
                  tooltip: 'AquaDoc Clinical Check',
                  onPressed: () => context.go('/aquadoc'),
                  icon: Icon(Icons.psychology, color: Theme.of(context).colorScheme.primary, size: 18),
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
        color: Colors.white.withValues(alpha: 0.15),
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
          style: TextStyle(fontSize: 10.5, color: Colors.grey[400], fontWeight: FontWeight.w500),
        ),
        const SizedBox(height: 2),
        Text(
          value,
          style: TextStyle(fontSize: 12, color: color, fontWeight: FontWeight.bold),
        ),
      ],
    );
  }
}
