import 'dart:math' as math;
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

  // Tank Dimensions (Meters)
  double _tankLengthM = 15.0;
  double _tankWidthM = 9.0;
  double _tankDepthM = 1.5;

  // Stocking & Production Timeline
  final String _species = 'African Catfish (Clarias gariepinus)';
  int _population = 4850;
  int _productionDay = 85; // Day 1 to 180
  double _initialWeightG = 320.0;
  final double _targetWeightG = 800.0;

  // Water Quality Parameters
  double _waterTempC = 28.5;
  double _dissolvedOxygenMgL = 5.8;
  double _ammoniaTanMgL = 0.15;
  final int _horizonDays = 60;

  // Tank Calculations
  double get _tankVolumeM3 => _tankLengthM * _tankWidthM * _tankDepthM;
  double get _tankSurfaceAreaM2 => _tankLengthM * _tankWidthM;

  // Biological Stress & Mortality Models
  double get _hypoxiaMortalityDaily {
    if (_dissolvedOxygenMgL >= 4.5) return 0.0;
    if (_dissolvedOxygenMgL >= 3.0) return 0.05;
    if (_dissolvedOxygenMgL >= 2.5) return 0.40;
    return 2.60; // Acute lethal hypoxia (<2.5 mg/L)
  }

  double get _ammoniaMortalityDaily {
    if (_ammoniaTanMgL <= 0.5) return 0.0;
    if (_ammoniaTanMgL <= 1.5) return 0.06;
    if (_ammoniaTanMgL <= 2.5) return 0.35;
    return 1.85; // Toxic ammonia gill damage (>2.5 mg/L)
  }

  double get _tempMortalityDaily {
    if (_waterTempC >= 25.0 && _waterTempC <= 31.0) return 0.0;
    if ((_waterTempC >= 22.0 && _waterTempC < 25.0) || (_waterTempC > 31.0 && _waterTempC <= 34.0)) {
      return 0.08;
    }
    return 0.45; // Extreme temperature stupor/shock
  }

  double get _densityKgM3 {
    final vol = _tankVolumeM3;
    if (vol <= 0) return 0.0;
    return ((_population * _initialWeightG) / 1000.0) / vol;
  }

  double get _densityMortalityDaily {
    if (_densityKgM3 <= 45.0) return 0.0;
    if (_densityKgM3 <= 75.0) return 0.12;
    return 0.60; // Overcrowding hypoxia/stress
  }

  // Predictive Survival Rate & Surviving Population
  double get _calculatedSurvivalRate {
    const baseDailyLoss = 0.015;
    final totalDailyLossPercent =
        baseDailyLoss + _hypoxiaMortalityDaily + _ammoniaMortalityDaily + _tempMortalityDaily + _densityMortalityDaily;
    final dailyRetention = 1.0 - (totalDailyLossPercent / 100.0);
    final cumulativeSurvival = math.pow(dailyRetention, _productionDay).toDouble() * 100.0;
    return cumulativeSurvival.clamp(5.0, 99.0);
  }

  int get _survivingStock => (_population * (_calculatedSurvivalRate / 100.0)).round();
  int get _mortalityCount => _population - _survivingStock;
  double get _currentBiomassKg => (_survivingStock * _initialWeightG) / 1000.0;

  // SGR & Growth Calculations
  double get _q10Factor => (1.0 + (_waterTempC - 28.0) * 0.05).clamp(0.6, 1.4);
  bool get _isDOInterlocked => _dissolvedOxygenMgL < 3.0;
  bool get _isTanStressed => _ammoniaTanMgL > 2.0;

  double get _effectiveSGR {
    if (_isDOInterlocked) return 0.0;
    double baseSGR = 2.45 * _q10Factor;
    if (_isTanStressed) baseSGR *= 0.5;
    if (_dissolvedOxygenMgL < 4.0) baseSGR *= 0.75;
    if (_densityKgM3 > 60.0) baseSGR *= 0.85;
    return baseSGR.clamp(0.1, 3.8);
  }

  int get _daysToHarvest {
    if (_effectiveSGR <= 0) return 999;
    final remainingWeightRatio = (_targetWeightG / _initialWeightG).clamp(1.01, 20.0);
    final days = (100 * math.log(remainingWeightRatio) / _effectiveSGR).round();
    return days.clamp(10, 365);
  }

  double get _projectedHarvestBiomassKg {
    final currentBiomass = _currentBiomassKg;
    if (_isDOInterlocked) return currentBiomass;
    final weightGainG = _initialWeightG * (_effectiveSGR / 100.0) * _horizonDays;
    final finalAvgWeightG = (_initialWeightG + weightGainG).clamp(_initialWeightG, _targetWeightG * 1.5);
    final harvestSurviving = (_survivingStock * 0.98).round();
    return (harvestSurviving * finalAvgWeightG) / 1000.0;
  }

  double get _estimatedFeedNeededKg {
    final biomassGain = _projectedHarvestBiomassKg - _currentBiomassKg;
    return (biomassGain * 1.18).clamp(0.0, 50000.0);
  }

  String get _productionPhaseName {
    if (_productionDay <= 30) return '🌱 Fingerling Stage (Day 1-30)';
    if (_productionDay <= 70) return '🐟 Juvenile Stage (Day 31-70)';
    if (_productionDay <= 130) return '🚀 Main Growout Stage (Day 71-130)';
    return '🏆 Finishing / Market Harvest Stage';
  }

  void _applyPreset(String key) {
    setState(() {
      _selectedPreset = key;
      switch (key) {
        case 'optimal':
          _waterTempC = 28.5;
          _dissolvedOxygenMgL = 5.8;
          _ammoniaTanMgL = 0.15;
          _tankLengthM = 15.0;
          _tankWidthM = 9.0;
          _tankDepthM = 1.5;
          break;
        case 'hypoxia':
          _waterTempC = 29.0;
          _dissolvedOxygenMgL = 2.3; // Interlock trigger + severe mortality
          _ammoniaTanMgL = 0.35;
          break;
        case 'ammonia':
          _waterTempC = 28.2;
          _dissolvedOxygenMgL = 4.8;
          _ammoniaTanMgL = 2.8; // High TAN! Chemical stress
          break;
        case 'cold':
          _waterTempC = 20.5; // Cold slowdown
          _dissolvedOxygenMgL = 6.4;
          _ammoniaTanMgL = 0.10;
          break;
        case 'ras_dense':
          _population = 7500;
          _tankLengthM = 10.0;
          _tankWidthM = 6.0;
          _tankDepthM = 1.8;
          _waterTempC = 28.0;
          _dissolvedOxygenMgL = 7.0;
          _ammoniaTanMgL = 0.40;
          break;
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final sgr = _effectiveSGR;
    final harvestDays = _daysToHarvest;
    final projectedBiomass = _projectedHarvestBiomassKg;
    final feedNeeded = _estimatedFeedNeededKg;
    final isBlocked = _isDOInterlocked;
    final survivalRate = _calculatedSurvivalRate;
    final surviving = _survivingStock;
    final mortalities = _mortalityCount;
    final density = _densityKgM3;

    final isDark = Theme.of(context).brightness == Brightness.dark;

    return Scaffold(
      appBar: AppBar(
        title: const Text('Farm Environmental & Growth Simulator'),
        actions: [
          IconButton(
            icon: const Icon(Icons.psychology),
            tooltip: 'Consult AquaDoc AI',
            onPressed: () {
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(
                  content: Text('Simulated pond parameters synced to AquaDoc!'),
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
            style: Theme.of(context).textTheme.titleSmall?.copyWith(
                  fontWeight: FontWeight.bold,
                  color: isDark ? Colors.grey[400] : const Color(0xFF4A5568),
                ),
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
                  label: '🚨 Hypoxia Crisis (DO < 3)',
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
          const SizedBox(height: 18),

          // Primary Projection & Interlock Banner
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
                  color: (isBlocked ? Colors.red : Theme.of(context).colorScheme.primary).withValues(alpha: 0.3),
                  blurRadius: 12,
                  offset: const Offset(0, 5),
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
                      isBlocked ? 'DETERMINISTIC INTERLOCK: FEEDING BLOCKED' : _productionPhaseName.toUpperCase(),
                      style: const TextStyle(
                        color: Colors.white70,
                        fontSize: 10.5,
                        fontWeight: FontWeight.bold,
                        letterSpacing: 1.0,
                      ),
                    ),
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                      decoration: BoxDecoration(
                        color: Colors.white.withValues(alpha: 0.2),
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
                        const Text('Harvest Biomass Projection', style: TextStyle(color: Colors.white70, fontSize: 12)),
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
                        const Text('Days to 800g Target', style: TextStyle(color: Colors.white70, fontSize: 12)),
                        const SizedBox(height: 2),
                        Text(
                          isBlocked ? 'PAUSED' : '$harvestDays days',
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
                      'Metabolic Q10: ${_q10Factor.toStringAsFixed(2)}x',
                      style: const TextStyle(color: Colors.white70, fontSize: 12),
                    ),
                  ],
                ),
              ],
            ),
          ),
          const SizedBox(height: 18),

          // 3D Digital Twin Tank Visualizer reflecting wide dimensions & mortality
          DigitalTwin3DVisualizer(
            dissolvedOxygen: _dissolvedOxygenMgL,
            temperature: _waterTempC,
            ammoniaTan: _ammoniaTanMgL,
            avgWeightG: _initialWeightG,
            biomassKg: _currentBiomassKg,
            population: _population,
            tankLengthM: _tankLengthM,
            tankWidthM: _tankWidthM,
            tankDepthM: _tankDepthM,
            productionPeriodDays: _productionDay,
            survivalRate: survivalRate,
            mortalityCount: mortalities,
            species: _species,
          ),
          const SizedBox(height: 18),

          // MORTALITY & SURVIVAL PREDICTION CARD
          Card(
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(18),
              side: BorderSide(
                color: survivalRate < 85.0
                    ? Colors.redAccent.withValues(alpha: 0.6)
                    : (survivalRate < 93.0 ? Colors.amberAccent.withValues(alpha: 0.4) : Colors.greenAccent.withValues(alpha: 0.4)),
                width: 1.5,
              ),
            ),
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      Row(
                        children: [
                          Icon(
                            Icons.favorite,
                            color: survivalRate < 85.0
                                ? Colors.redAccent
                                : (survivalRate < 93.0 ? Colors.amberAccent : Colors.greenAccent),
                            size: 20,
                          ),
                          const SizedBox(width: 8),
                          const Text(
                            'Mortality & Survival Prediction',
                            style: TextStyle(fontWeight: FontWeight.bold, fontSize: 15),
                          ),
                        ],
                      ),
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                        decoration: BoxDecoration(
                          color: (survivalRate < 85.0
                                  ? Colors.red
                                  : (survivalRate < 93.0 ? Colors.amber : Colors.green))
                              .withValues(alpha: 0.15),
                          borderRadius: BorderRadius.circular(12),
                        ),
                        child: Text(
                          '${survivalRate.toStringAsFixed(1)}% Survival',
                          style: TextStyle(
                            color: survivalRate < 85.0
                                ? Colors.redAccent
                                : (survivalRate < 93.0 ? Colors.amberAccent : Colors.greenAccent),
                            fontWeight: FontWeight.bold,
                            fontSize: 13,
                          ),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 16),
                  Row(
                    children: [
                      Expanded(
                        child: _MiniMetricTile(
                          label: 'Surviving Stock',
                          value: '$surviving fish',
                          subtext: '${(survivalRate).toStringAsFixed(1)}% of original set',
                          color: Colors.greenAccent,
                        ),
                      ),
                      const SizedBox(width: 10),
                      Expanded(
                        child: _MiniMetricTile(
                          label: 'Predicted Mortality',
                          value: '$mortalities fish',
                          subtext: 'Cumulative losses',
                          color: mortalities > 200 ? Colors.redAccent : Colors.orangeAccent,
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 10),
                  Row(
                    children: [
                      Expanded(
                        child: _MiniMetricTile(
                          label: 'Stocking Density',
                          value: '${density.toStringAsFixed(1)} kg/m³',
                          subtext: density > 60 ? '⚠️ High Density Alert' : '✅ Optimal Range',
                          color: density > 60 ? Colors.amberAccent : Colors.cyanAccent,
                        ),
                      ),
                      const SizedBox(width: 10),
                      Expanded(
                        child: _MiniMetricTile(
                          label: 'Current Biomass',
                          value: '${_currentBiomassKg.toStringAsFixed(0)} kg',
                          subtext: 'Day $_productionDay standing crop',
                          color: Colors.tealAccent,
                        ),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 18),

          // TANK DIMENSIONS & GEOMETRY SUMMARY
          Card(
            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(18)),
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Row(
                    children: [
                      Icon(Icons.aspect_ratio, color: Colors.cyanAccent, size: 20),
                      SizedBox(width: 8),
                      Text('Culture Tank Dimensions & Capacity', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 15)),
                    ],
                  ),
                  const SizedBox(height: 14),
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceAround,
                    children: [
                      _DimensionPill(label: 'Length', value: '${_tankLengthM.toStringAsFixed(1)} m'),
                      _DimensionPill(label: 'Width', value: '${_tankWidthM.toStringAsFixed(1)} m'),
                      _DimensionPill(label: 'Depth', value: '${_tankDepthM.toStringAsFixed(1)} m'),
                      _DimensionPill(label: 'Volume', value: '${_tankVolumeM3.toStringAsFixed(0)} m³'),
                      _DimensionPill(label: 'Area', value: '${_tankSurfaceAreaM2.toStringAsFixed(0)} m²'),
                    ],
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 20),

          // INTERACTIVE SLIDERS SECTION
          Text(
            'Interactive Parameter Controls',
            style: Theme.of(context).textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.bold,
                  color: isDark ? Colors.white : const Color(0xFF1A202C),
                ),
          ),
          const SizedBox(height: 12),

          // Production Period Slider
          _SliderCard(
            title: 'Production Period (Day in Cycle)',
            value: _productionDay.toDouble(),
            min: 1,
            max: 180,
            divisions: 179,
            unit: 'Days',
            subtext: _productionPhaseName,
            onChanged: (v) => setState(() => _productionDay = v.toInt()),
          ),
          const SizedBox(height: 10),

          // Tank Length Slider
          _SliderCard(
            title: 'Tank Length (Meters)',
            value: _tankLengthM,
            min: 4.0,
            max: 35.0,
            divisions: 62,
            unit: 'm',
            onChanged: (v) => setState(() => _tankLengthM = v),
          ),
          const SizedBox(height: 10),

          // Tank Width Slider
          _SliderCard(
            title: 'Tank Width (Meters)',
            value: _tankWidthM,
            min: 3.0,
            max: 20.0,
            divisions: 34,
            unit: 'm',
            onChanged: (v) => setState(() => _tankWidthM = v),
          ),
          const SizedBox(height: 10),

          // Tank Depth Slider
          _SliderCard(
            title: 'Water Depth (Meters)',
            value: _tankDepthM,
            min: 0.8,
            max: 2.5,
            divisions: 17,
            unit: 'm',
            onChanged: (v) => setState(() => _tankDepthM = v),
          ),
          const SizedBox(height: 10),

          // Population Slider
          _SliderCard(
            title: 'Initial Stocking Population',
            value: _population.toDouble(),
            min: 500,
            max: 10000,
            divisions: 95,
            unit: 'fish',
            subtext: 'Accurate stock set will reflect live in twin simulation',
            onChanged: (v) => setState(() => _population = v.toInt()),
          ),
          const SizedBox(height: 10),

          // Initial Weight Slider
          _SliderCard(
            title: 'Current Average Weight (g)',
            value: _initialWeightG,
            min: 10.0,
            max: 900.0,
            divisions: 89,
            unit: 'g',
            onChanged: (v) => setState(() => _initialWeightG = v),
          ),
          const SizedBox(height: 10),

          // DO Slider
          _SliderCard(
            title: 'Dissolved Oxygen (mg/L)',
            value: _dissolvedOxygenMgL,
            min: 1.5,
            max: 9.0,
            divisions: 75,
            unit: 'mg/L',
            isWarning: _dissolvedOxygenMgL < 3.0,
            warningText: 'DO < 3.0 mg/L strictly blocks feeding interlock & causes hypoxia mortality',
            onChanged: (v) => setState(() => _dissolvedOxygenMgL = v),
          ),
          const SizedBox(height: 10),

          // Temperature Slider
          _SliderCard(
            title: 'Water Temperature (°C)',
            value: _waterTempC,
            min: 16.0,
            max: 36.0,
            divisions: 40,
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
            divisions: 69,
            unit: 'mg/L',
            isWarning: _ammoniaTanMgL > 2.0,
            warningText: 'TAN > 2.0 mg/L triggers toxic gill stress and cuts ration by 50%',
            onChanged: (v) => setState(() => _ammoniaTanMgL = v),
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
          const SizedBox(height: 24),
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
      label: Text(
        label,
        style: TextStyle(
          fontSize: 12,
          fontWeight: isSelected ? FontWeight.bold : FontWeight.normal,
        ),
      ),
      selected: isSelected,
      selectedColor: isAlert
          ? Colors.red.withValues(alpha: 0.25)
          : Theme.of(context).colorScheme.primary.withValues(alpha: 0.25),
      checkmarkColor: isAlert ? Colors.redAccent : Theme.of(context).colorScheme.primary,
      onSelected: (_) => onTap(),
    );
  }
}

class _MiniMetricTile extends StatelessWidget {
  final String label;
  final String value;
  final String subtext;
  final Color color;

  const _MiniMetricTile({
    required this.label,
    required this.value,
    required this.subtext,
    required this.color,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.10),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: color.withValues(alpha: 0.25)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(label, style: const TextStyle(fontSize: 11, color: Colors.grey, fontWeight: FontWeight.w500)),
          const SizedBox(height: 3),
          Text(
            value,
            style: TextStyle(fontSize: 15, fontWeight: FontWeight.bold, color: color),
          ),
          const SizedBox(height: 2),
          Text(subtext, style: const TextStyle(fontSize: 9.5, color: Colors.grey)),
        ],
      ),
    );
  }
}

class _DimensionPill extends StatelessWidget {
  final String label;
  final String value;

  const _DimensionPill({required this.label, required this.value});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Text(label, style: const TextStyle(fontSize: 10, color: Colors.grey)),
        const SizedBox(height: 2),
        Text(value, style: const TextStyle(fontSize: 13, fontWeight: FontWeight.bold, color: Colors.cyanAccent)),
      ],
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
  final String? subtext;
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
    this.subtext,
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
                Expanded(
                  child: Text(
                    title,
                    style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 13),
                  ),
                ),
                Text(
                  '${value is int || value == value.roundToDouble() ? value.toInt() : value.toStringAsFixed(1)} $unit',
                  style: TextStyle(
                    fontWeight: FontWeight.bold,
                    fontSize: 14,
                    color: isWarning ? Colors.red : Theme.of(context).colorScheme.primary,
                  ),
                ),
              ],
            ),
            if (subtext != null) ...[
              const SizedBox(height: 2),
              Text(
                subtext!,
                style: const TextStyle(fontSize: 11, color: Colors.grey),
              ),
            ],
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
