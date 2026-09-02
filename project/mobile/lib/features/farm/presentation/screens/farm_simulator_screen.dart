import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/theme/app_theme.dart';
import '../widgets/digital_twin_3d_visualizer.dart';

class FarmSimulatorScreen extends ConsumerStatefulWidget {
  const FarmSimulatorScreen({super.key});

  @override
  ConsumerState<FarmSimulatorScreen> createState() => _FarmSimulatorScreenState();
}

class _FarmSimulatorScreenState extends ConsumerState<FarmSimulatorScreen> {
  // Scenario Preset
  String _selectedPreset = 'optimal';

  // Parameters
  String _species = 'Clarias Catfish';
  int _population = 2000;
  double _initialWeightG = 120.0;
  double _targetWeightG = 800.0;
  double _waterTempC = 28.5;
  double _dissolvedOxygenMgL = 5.5;
  double _ammoniaTanMgL = 0.20;
  double _dailyRationG = 4500.0;
  int _horizonDays = 60;

  // Simulation Calculations
  double get _q10Factor => (1.0 + (_waterTempC - 28.0) * 0.05).clamp(0.6, 1.4);
  bool get _isDOInterlocked => _dissolvedOxygenMgL < 3.0;
  bool get _isTanStressed => _ammoniaTanMgL > 2.0;

  // SGR calculation: SGR = (ln(W_final) - ln(W_init)) / t
  double get _effectiveSGR {
    if (_isDOInterlocked) return 0.0; // Feeding blocked, no growth
    double baseSGR = 2.45 * _q10Factor;
    if (_isTanStressed) baseSGR *= 0.5; // Ammonia stress cuts growth by 50%
    if (_dissolvedOxygenMgL < 4.0) baseSGR *= 0.75;
    return baseSGR.clamp(0.2, 3.8);
  }

  // Projected days to reach target weight
  int get _daysToHarvest {
    if (_effectiveSGR <= 0) return 999;
    // W_t = W_0 * e^(SGR * t / 100)
    // t = 100 * ln(W_t / W_0) / SGR
    final days = (100 * (1.897) / _effectiveSGR).round();
    return days.clamp(15, 365);
  }

  double get _projectedBiomassKg {
    // Current Biomass
    final currentBiomass = (_population * _initialWeightG) / 1000.0;
    if (_isDOInterlocked) return currentBiomass;
    // Estimated harvest biomass at horizon
    final weightGainG = _initialWeightG * (_effectiveSGR / 100.0) * _horizonDays;
    final finalAvgWeightG = (_initialWeightG + weightGainG).clamp(_initialWeightG, _targetWeightG * 1.5);
    return (_population * 0.96 * finalAvgWeightG) / 1000.0; // 96% survival
  }

  double get _estimatedFeedNeededKg {
    final biomassGain = _projectedBiomassKg - ((_population * _initialWeightG) / 1000.0);
    return (biomassGain * 1.18).clamp(0.0, 50000.0); // 1.18 FCR
  }

  void _applyPreset(String key) {
    setState(() {
      _selectedPreset = key;
      switch (key) {
        case 'optimal':
          _waterTempC = 28.5;
          _dissolvedOxygenMgL = 5.8;
          _ammoniaTanMgL = 0.15;
          break;
        case 'hypoxia':
          _waterTempC = 29.0;
          _dissolvedOxygenMgL = 2.4; // Safety Interlock Trigger!
          _ammoniaTanMgL = 0.35;
          break;
        case 'ammonia':
          _waterTempC = 28.2;
          _dissolvedOxygenMgL = 4.8;
          _ammoniaTanMgL = 2.6; // High TAN! 50% ration cut
          break;
        case 'cold':
          _waterTempC = 21.0; // Cold slowdown
          _dissolvedOxygenMgL = 6.4;
          _ammoniaTanMgL = 0.10;
          break;
        case 'ras_dense':
          _population = 6000;
          _waterTempC = 28.0;
          _dissolvedOxygenMgL = 6.8;
          _ammoniaTanMgL = 0.80;
          break;
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final sgr = _effectiveSGR;
    final harvestDays = _daysToHarvest;
    final projectedBiomass = _projectedBiomassKg;
    final feedNeeded = _estimatedFeedNeededKg;
    final isBlocked = _isDOInterlocked;
    final isTanWarn = _isTanStressed;

    return Scaffold(
      appBar: AppBar(
        title: const Text('Farm Environmental Simulator'),
        actions: [
          IconButton(
            icon: const Icon(Icons.psychology),
            tooltip: 'Consult AquaDoc with Scenario',
            onPressed: () {
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(
                  content: Text('Simulated pond state injected into AquaDoc!'),
                  backgroundColor: AppTheme.deviceOnline,
                ),
              );
              context.go('/aquadoc');
            },
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          // Scenario Preset Chips
          Text(
            'Preset Scenarios',
            style: Theme.of(context).textTheme.titleSmall?.copyWith(fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 8),
          SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: Row(
              children: [
                _PresetChip(
                  label: '🌟 Optimal Growout',
                  isSelected: _selectedPreset == 'optimal',
                  onTap: () => _applyPreset('optimal'),
                ),
                const SizedBox(width: 8),
                _PresetChip(
                  label: '⚠️ Hypoxia Crisis (DO < 3)',
                  isSelected: _selectedPreset == 'hypoxia',
                  onTap: () => _applyPreset('hypoxia'),
                  isAlert: true,
                ),
                const SizedBox(width: 8),
                _PresetChip(
                  label: '🧪 High Ammonia (TAN > 2)',
                  isSelected: _selectedPreset == 'ammonia',
                  onTap: () => _applyPreset('ammonia'),
                  isAlert: true,
                ),
                const SizedBox(width: 8),
                _PresetChip(
                  label: '❄️ Cold Water (< 22°C)',
                  isSelected: _selectedPreset == 'cold',
                  onTap: () => _applyPreset('cold'),
                ),
                const SizedBox(width: 8),
                _PresetChip(
                  label: '⚡ Ultra-Dense RAS',
                  isSelected: _selectedPreset == 'ras_dense',
                  onTap: () => _applyPreset('ras_dense'),
                ),
              ],
            ),
          ),
          const SizedBox(height: 20),

          // Real-Time Projection Card
          Container(
            padding: const EdgeInsets.all(18),
            decoration: BoxDecoration(
              gradient: LinearGradient(
                colors: isBlocked
                    ? [Colors.red[900]!, Colors.red[800]!]
                    : [Theme.of(context).colorScheme.primary, const Color(0xFF1E3C72)],
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
              ),
              borderRadius: BorderRadius.circular(20),
              boxShadow: [
                BoxShadow(
                  color: (isBlocked ? Colors.red : Theme.of(context).colorScheme.primary).withOpacity(0.3),
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
                    Text(
                      isBlocked ? 'FEEDING INTERLOCK BLOCKED' : 'PROJECTION: $_horizonDays DAYS HORIZON',
                      style: const TextStyle(
                        color: Colors.white70,
                        fontSize: 11,
                        fontWeight: FontWeight.bold,
                        letterSpacing: 1.1,
                      ),
                    ),
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                      decoration: BoxDecoration(
                        color: Colors.white.withOpacity(0.2),
                        borderRadius: BorderRadius.circular(10),
                      ),
                      child: Text(
                        'SGR: ${sgr.toStringAsFixed(2)} %/day',
                        style: const TextStyle(color: Colors.white, fontSize: 11, fontWeight: FontWeight.bold),
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const Text('Projected Biomass', style: TextStyle(color: Colors.white70, fontSize: 12)),
                        const SizedBox(height: 2),
                        Text(
                          '${projectedBiomass.toStringAsFixed(1)} kg',
                          style: const TextStyle(color: Colors.white, fontSize: 24, fontWeight: FontWeight.bold),
                        ),
                      ],
                    ),
                    Column(
                      crossAxisAlignment: CrossAxisAlignment.end,
                      children: [
                        const Text('Days to 800g Harvest', style: TextStyle(color: Colors.white70, fontSize: 12)),
                        const SizedBox(height: 2),
                        Text(
                          isBlocked ? 'BLOCKED' : '$harvestDays days',
                          style: const TextStyle(color: Colors.white, fontSize: 24, fontWeight: FontWeight.bold),
                        ),
                      ],
                    ),
                  ],
                ),
                const SizedBox(height: 14),
                const Divider(color: Colors.white24, height: 1),
                const SizedBox(height: 12),
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Text(
                      'Feed Needed: ${feedNeeded.toStringAsFixed(0)} kg (1.18 FCR)',
                      style: const TextStyle(color: Colors.white70, fontSize: 12),
                    ),
                    Text(
                      'Q10 Factor: ${_q10Factor.toStringAsFixed(2)}x',
                      style: const TextStyle(color: Colors.white70, fontSize: 12),
                    ),
                  ],
                ),
              ],
            ),
          ),
          const SizedBox(height: 20),

          // 3D Animated Culture Tank Visualizer responding live to sliders
          DigitalTwin3DVisualizer(
            dissolvedOxygen: _dissolvedOxygenMgL,
            temperature: _waterTempC,
            ammoniaTan: _ammoniaTanMgL,
            avgWeightG: _initialWeightG,
            biomassKg: (_population * _initialWeightG) / 1000.0,
            population: _population,
          ),
          const SizedBox(height: 20),

          // Parameter Sliders Group
          Text(
            'Interactive Parameter Controls',
            style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 12),

          // DO Slider
          _SliderCard(
            title: 'Dissolved Oxygen (mg/L)',
            value: _dissolvedOxygenMgL,
            min: 1.5,
            max: 9.0,
            unit: 'mg/L',
            isWarning: _dissolvedOxygenMgL < 3.0,
            warningText: 'DO < 3.0 mg/L strictly blocks feeding interlock',
            onChanged: (v) => setState(() => _dissolvedOxygenMgL = v),
          ),
          const SizedBox(height: 10),

          // Temperature Slider
          _SliderCard(
            title: 'Water Temperature (°C)',
            value: _waterTempC,
            min: 16.0,
            max: 36.0,
            unit: '°C',
            isWarning: _waterTempC < 22.0 || _waterTempC > 33.0,
            warningText: 'Optimal Clarias growth is 26°C - 30°C',
            onChanged: (v) => setState(() => _waterTempC = v),
          ),
          const SizedBox(height: 10),

          // Ammonia TAN Slider
          _SliderCard(
            title: 'Ammonia TAN (mg/L)',
            value: _ammoniaTanMgL,
            min: 0.05,
            max: 3.5,
            unit: 'mg/L',
            isWarning: _ammoniaTanMgL > 2.0,
            warningText: 'TAN > 2.0 mg/L reduces feeding ration by 50%',
            onChanged: (v) => setState(() => _ammoniaTanMgL = v),
          ),
          const SizedBox(height: 10),

          // Population Slider
          _SliderCard(
            title: 'Stocking Population (Fish Count)',
            value: _population.toDouble(),
            min: 500,
            max: 10000,
            divisions: 95,
            unit: 'fish',
            onChanged: (v) => setState(() => _population = v.toInt()),
          ),
          const SizedBox(height: 10),

          // Initial Weight Slider
          _SliderCard(
            title: 'Current Average Weight (g)',
            value: _initialWeightG,
            min: 10.0,
            max: 500.0,
            unit: 'g',
            onChanged: (v) => setState(() => _initialWeightG = v),
          ),
          const SizedBox(height: 24),

          // Action Buttons
          Row(
            children: [
              Expanded(
                child: FilledButton.icon(
                  onPressed: () {
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(
                        content: Text('Simulated digital twin synced to AquaDoc & farm models!'),
                        backgroundColor: AppTheme.deviceOnline,
                      ),
                    );
                    context.go('/twin');
                  },
                  icon: const Icon(Icons.sync),
                  label: const Text('View in AquaTwin'),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: () => context.go('/aquadoc'),
                  icon: const Icon(Icons.psychology),
                  label: const Text('Ask AquaDoc AI'),
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

class _PresetChip extends StatelessWidget {
  final String label;
  final bool isSelected;
  final VoidCallback onTap;
  final bool isAlert;

  const _PresetChip({
    required this.label,
    required this.isSelected,
    required this.onTap,
    this.isAlert = false,
  });

  @override
  Widget build(BuildContext context) {
    return FilterChip(
      label: Text(label, style: TextStyle(fontSize: 12, fontWeight: isSelected ? FontWeight.bold : FontWeight.normal)),
      selected: isSelected,
      selectedColor: isAlert ? Colors.red.withOpacity(0.2) : Theme.of(context).colorScheme.primary.withOpacity(0.2),
      checkmarkColor: isAlert ? Colors.red : Theme.of(context).colorScheme.primary,
      onSelected: (_) => onTap(),
    );
  }
}

class _SliderCard extends StatelessWidget {
  final String title;
  final double value;
  final double min;
  final double max;
  final int? divisions;
  final String unit;
  final bool isWarning;
  final String? warningText;
  final ValueChanged<double> onChanged;

  const _SliderCard({
    required this.title,
    required this.value,
    required this.min,
    required this.max,
    this.divisions,
    required this.unit,
    this.isWarning = false,
    this.warningText,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      elevation: 1,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(14),
        side: isWarning ? const BorderSide(color: Colors.red, width: 1.2) : BorderSide.none,
      ),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(title, style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 13)),
                Text(
                  '${value is int ? value.toInt() : value.toStringAsFixed(1)} $unit',
                  style: TextStyle(
                    fontWeight: FontWeight.bold,
                    fontSize: 14,
                    color: isWarning ? Colors.red : Theme.of(context).colorScheme.primary,
                  ),
                ),
              ],
            ),
            Slider(
              value: value.clamp(min, max),
              min: min,
              max: max,
              divisions: divisions ?? 50,
              onChanged: onChanged,
            ),
            if (isWarning && warningText != null)
              Padding(
                padding: const EdgeInsets.only(left: 4, bottom: 4),
                child: Row(
                  children: [
                    const Icon(Icons.warning_amber, color: Colors.red, size: 14),
                    const SizedBox(width: 4),
                    Expanded(
                      child: Text(
                        warningText!,
                        style: const TextStyle(fontSize: 11, color: Colors.red, fontWeight: FontWeight.w500),
                      ),
                    ),
                  ],
                ),
              ),
          ],
        ),
      ),
    );
  }
}
