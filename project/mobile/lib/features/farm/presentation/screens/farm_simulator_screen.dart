import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/models/farm_unit.dart';
import '../../../../core/providers/farm_provider.dart';
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

  void _importFromUnit(FarmUnit unit) {
    setState(() {
      _selectedPreset = 'custom';
      _tankLengthM = unit.lengthM;
      _tankWidthM = unit.widthM;
      _tankDepthM = unit.depthM;
      _population = unit.fishCount;
      _initialWeightG = unit.avgWeightGrams;
      _productionDay = unit.daysInProduction.clamp(1, 180);
      if (unit.manualTemp != null) _waterTempC = unit.manualTemp!;
      if (unit.manualDO != null) _dissolvedOxygenMgL = unit.manualDO!;
      if (unit.manualTAN != null) _ammoniaTanMgL = unit.manualTAN!;
    });
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text('Imported "${unit.name}" (${unit.lengthM}m × ${unit.widthM}m, ${unit.fishCount} fish) into Simulator!'),
        backgroundColor: AppTheme.deviceOnline,
      ),
    );
  }

  String _generatePlanReportText() {
    final vol = _tankVolumeM3;
    final harvestSurv = _survivingStock;
    final harvestBio = _projectedHarvestBiomassKg;
    final feed = _estimatedFeedNeededKg;
    final bags = (feed / 15.0).ceil();
    final survRate = _calculatedSurvivalRate;
    final initDensity = _densityKgM3;
    final harvDensity = vol > 0 ? (harvestBio / vol) : 0.0;

    return '''
========================================================
   SMARTAQUA FARMER PRODUCTION PLAN & BIOENERGETICS
========================================================
Report Generated: ${DateTime.now().toLocal().toString().split('.')[0]}
Cultured Species: $_species
Target Cycle Duration: 180 Days (Current Timeline: Day $_productionDay)
Stage: $_productionPhaseName

--------------------------------------------------------
1. FACILITY GEOMETRY & STOCKING CAPACITY
--------------------------------------------------------
- Tank Dimensions: ${_tankLengthM.toStringAsFixed(1)}m (L) × ${_tankWidthM.toStringAsFixed(1)}m (W) × ${_tankDepthM.toStringAsFixed(1)}m (Depth)
- Water Volume: ${vol.toStringAsFixed(1)} m³ (${(vol * 1000).toStringAsFixed(0)} Litres)
- Water Surface Area: ${_tankSurfaceAreaM2.toStringAsFixed(1)} m²
- Initial Stock Population: $_population fish
- Initial Average Weight: ${_initialWeightG.toStringAsFixed(0)} g
- Initial Standing Biomass: ${_currentBiomassKg.toStringAsFixed(1)} kg
- Initial Stocking Density: ${initDensity.toStringAsFixed(1)} kg/m³
- Projected Harvest Density: ${harvDensity.toStringAsFixed(1)} kg/m³
- Aeration Recommendation: ${harvDensity > 30 ? "HIGH DENSITY: Continuous mechanical diffuser aeration required (min 1.5 HP/1000kg)." : "STANDARD DENSITY: Night/Dawn aeration recommended during peak growout."}

--------------------------------------------------------
2. HARVEST & MORTALITY PROJECTIONS
--------------------------------------------------------
- Projected Survival Rate: ${survRate.toStringAsFixed(1)}%
- Estimated Surviving Fish at Harvest: $harvestSurv fish
- Cumulative Mortality Budget: $_mortalityCount fish
- Target Harvest Weight: ${_targetWeightG.toStringAsFixed(0)} g / fish
- Total Projected Harvest Biomass: ${harvestBio.toStringAsFixed(1)} kg (${(harvestBio / 1000).toStringAsFixed(2)} Metric Tons)
- Days to Target Weight: $_daysToHarvest days (at current ${_effectiveSGR.toStringAsFixed(2)}%/day SGR)

--------------------------------------------------------
3. FEED BUDGET & 4-PHASE PELLET SIZING SCHEDULE
--------------------------------------------------------
- Phase 1 (Days 1–30, 10g → 50g):
  * Pellet Size: 1.5mm – 2.0mm Micro-Pellets
  * Protein Level: 45% Crude Protein
  * Daily Ration: 6.0% of Body Weight (4 feeds/day)
- Phase 2 (Days 31–70, 50g → 180g):
  * Pellet Size: 3.0mm Extruded Floating
  * Protein Level: 40% Crude Protein
  * Daily Ration: 4.5% of Body Weight (3 feeds/day)
- Phase 3 (Days 71–130, 180g → 500g):
  * Pellet Size: 4.5mm Grower Pellets
  * Protein Level: 38% Crude Protein
  * Daily Ration: 3.0% of Body Weight (2–3 feeds/day)
- Phase 4 (Days 131–180, 500g → 800g):
  * Pellet Size: 6.0mm Finisher Floating Pellets
  * Protein Level: 35% Crude Protein
  * Daily Ration: 2.0% of Body Weight (2 feeds/day)

- Total Commercial Feed Required: ${feed.toStringAsFixed(1)} kg
- Standard 15-kg Bag Count: $bags Bags
- Projected Feed Conversion Ratio (FCR): 1.18

--------------------------------------------------------
4. WATER QUALITY SAFEGUARDS & THRESHOLDS
--------------------------------------------------------
- Dissolved Oxygen (DO): Min ≥ 4.5 mg/L (Safety Cutoff < 3.0 mg/L)
- Water Temperature: 26°C – 30°C (Metabolic Q10 factor: ${_q10Factor.toStringAsFixed(2)}x)
- Total Ammonia Nitrogen (TAN): < 0.5 mg/L (Gill Stress Cutoff > 2.0 mg/L)
- Water pH: 6.8 – 8.0

--------------------------------------------------------
5. ECONOMIC PROJECTIONS
--------------------------------------------------------
- Estimated Farm-Gate Value (@ \$3.50/kg): \$${(harvestBio * 3.50).toStringAsFixed(2)}
- Estimated Feed Cost (@ \$1.80/kg): \$${(feed * 1.80).toStringAsFixed(2)}
- Estimated Gross Profit Margin: \$${((harvestBio * 3.50) - (feed * 1.80)).toStringAsFixed(2)}
========================================================
''';
  }

  void _showFarmerPlanReportSheet(BuildContext context) {
    final reportText = _generatePlanReportText();
    final harvestBio = _projectedHarvestBiomassKg;
    final feed = _estimatedFeedNeededKg;
    final bags = (feed / 15.0).ceil();
    final survRate = _calculatedSurvivalRate;
    final vol = _tankVolumeM3;
    final harvDensity = vol > 0 ? (harvestBio / vol) : 0.0;

    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: const Color(0xFF0F1B2B),
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
      ),
      builder: (ctx) => DraggableScrollableSheet(
        initialChildSize: 0.85,
        maxChildSize: 0.95,
        minChildSize: 0.5,
        expand: false,
        builder: (ctx, scrollController) => Padding(
          padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 16),
          child: ListView(
            controller: scrollController,
            children: [
              // Handle Bar
              Center(
                child: Container(
                  width: 44,
                  height: 5,
                  decoration: BoxDecoration(
                    color: Colors.grey[600],
                    borderRadius: BorderRadius.circular(10),
                  ),
                ),
              ),
              const SizedBox(height: 16),

              // Title Header
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Row(
                    children: [
                      Container(
                        padding: const EdgeInsets.all(10),
                        decoration: BoxDecoration(
                          gradient: AppTheme.primaryGradient,
                          borderRadius: BorderRadius.circular(12),
                        ),
                        child: const Icon(Icons.assignment, color: Colors.white, size: 22),
                      ),
                      const SizedBox(width: 12),
                      const Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            'Farmer Production Plan',
                            style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold, color: Colors.white),
                          ),
                          Text(
                            '180-Day Bioenergetic Cycle Roadmap',
                            style: TextStyle(fontSize: 11.5, color: Colors.grey),
                          ),
                        ],
                      ),
                    ],
                  ),
                  IconButton(
                    icon: const Icon(Icons.close, color: Colors.white70),
                    onPressed: () => Navigator.pop(ctx),
                  ),
                ],
              ),
              const SizedBox(height: 20),

              // Summary Highlight Card
              Container(
                padding: const EdgeInsets.all(16),
                decoration: BoxDecoration(
                  gradient: const LinearGradient(
                    colors: [Color(0xFF0052D4), Color(0xFF4364F7), Color(0xFF6FB1FC)],
                    begin: Alignment.topLeft,
                    end: Alignment.bottomRight,
                  ),
                  borderRadius: BorderRadius.circular(18),
                  boxShadow: [
                    BoxShadow(
                      color: Colors.blue.withValues(alpha: 0.3),
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
                          _species.toUpperCase(),
                          style: const TextStyle(color: Colors.white70, fontSize: 11, fontWeight: FontWeight.bold),
                        ),
                        Container(
                          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                          decoration: BoxDecoration(
                            color: Colors.white.withValues(alpha: 0.2),
                            borderRadius: BorderRadius.circular(10),
                          ),
                          child: Text(
                            '${survRate.toStringAsFixed(1)}% Survival',
                            style: const TextStyle(color: Colors.white, fontSize: 11, fontWeight: FontWeight.bold),
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 10),
                    Text(
                      '${harvestBio.toStringAsFixed(1)} kg Harvest',
                      style: const TextStyle(color: Colors.white, fontSize: 26, fontWeight: FontWeight.bold),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      'Initial Stock: $_population fish (${_initialWeightG.toStringAsFixed(0)}g) • Target: ${_targetWeightG.toStringAsFixed(0)}g table size in $_daysToHarvest days',
                      style: const TextStyle(color: Colors.white, fontSize: 12.5),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 18),

              // Section 1: Geometry & Carrying Capacity
              _ReportSectionCard(
                title: '1. Facility Geometry & Carrying Capacity',
                icon: Icons.aspect_ratio,
                iconColor: Colors.cyanAccent,
                rows: [
                  _ReportRow(label: 'Enclosure Dimensions', value: '${_tankLengthM.toStringAsFixed(1)}m × ${_tankWidthM.toStringAsFixed(1)}m × ${_tankDepthM.toStringAsFixed(1)}m'),
                  _ReportRow(label: 'Effective Water Volume', value: '${vol.toStringAsFixed(1)} m³ (${(vol * 1000).toStringAsFixed(0)} L)'),
                  _ReportRow(label: 'Initial Stocking Density', value: '${_densityKgM3.toStringAsFixed(1)} kg/m³'),
                  _ReportRow(label: 'Final Harvest Density', value: '${harvDensity.toStringAsFixed(1)} kg/m³', isHighlight: true),
                  _ReportRow(
                    label: 'Aeration Protocol',
                    value: harvDensity > 30 ? 'High Density (24/7 Diffusers Req.)' : 'Standard (Night/Dawn)',
                  ),
                ],
              ),
              const SizedBox(height: 14),

              // Section 2: 4-Phase Feeding Roadmap
              _ReportSectionCard(
                title: '2. 4-Phase Feeding & Feed Budget Plan',
                icon: Icons.restaurant_menu,
                iconColor: Colors.orangeAccent,
                rows: [
                  const _ReportRow(label: 'Phase 1 (Day 1-30, 10-50g)', value: '1.5-2mm (45% CP) @ 6% BW (4x/day)'),
                  const _ReportRow(label: 'Phase 2 (Day 31-70, 50-180g)', value: '3.0mm (40% CP) @ 4.5% BW (3x/day)'),
                  const _ReportRow(label: 'Phase 3 (Day 71-130, 180-500g)', value: '4.5mm (38% CP) @ 3% BW (2-3x/day)'),
                  const _ReportRow(label: 'Phase 4 (Day 131-180, 500-800g)', value: '6.0mm (35% CP) @ 2% BW (2x/day)'),
                  _ReportRow(label: 'Total Feed Required', value: '${feed.toStringAsFixed(1)} kg ($bags × 15-kg bags)', isHighlight: true),
                  const _ReportRow(label: 'Target FCR', value: '1.18'),
                ],
              ),
              const SizedBox(height: 14),

              // Section 3: Water Quality Thresholds
              _ReportSectionCard(
                title: '3. Water Quality Safety Guardrails',
                icon: Icons.science_outlined,
                iconColor: Colors.purpleAccent,
                rows: [
                  const _ReportRow(label: 'Dissolved Oxygen (DO)', value: '≥ 4.5 mg/L (Cutoff: <3.0 mg/L)'),
                  _ReportRow(label: 'Optimal Temperature', value: '26°C - 30°C (Q10: ${_q10Factor.toStringAsFixed(2)}x)'),
                  const _ReportRow(label: 'Ammonia (TAN)', value: '< 0.5 mg/L (Cutoff: >2.0 mg/L)'),
                  const _ReportRow(label: 'Target pH Window', value: '6.8 - 8.0'),
                ],
              ),
              const SizedBox(height: 14),

              // Section 4: Economic Projections
              _ReportSectionCard(
                title: '4. Economic & Revenue Forecast',
                icon: Icons.monetization_on_outlined,
                iconColor: Colors.greenAccent,
                rows: [
                  _ReportRow(label: 'Est. Gross Revenue (@ \$3.50/kg)', value: '\$${(harvestBio * 3.50).toStringAsFixed(2)}', isHighlight: true),
                  _ReportRow(label: 'Est. Feed Expenditure (@ \$1.80/kg)', value: '\$${(feed * 1.80).toStringAsFixed(2)}'),
                  _ReportRow(label: 'Projected Gross Operating Margin', value: '\$${((harvestBio * 3.50) - (feed * 1.80)).toStringAsFixed(2)}', isHighlight: true),
                ],
              ),
              const SizedBox(height: 20),

              // Action Buttons
              Row(
                children: [
                  Expanded(
                    child: FilledButton.icon(
                      style: FilledButton.styleFrom(
                        backgroundColor: AppTheme.primaryCyan,
                        foregroundColor: const Color(0xFF0A192F),
                        padding: const EdgeInsets.symmetric(vertical: 14),
                        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
                      ),
                      onPressed: () {
                        Clipboard.setData(ClipboardData(text: reportText));
                        ScaffoldMessenger.of(context).showSnackBar(
                          const SnackBar(
                            content: Text('Copied Farmer Production Plan Report to clipboard!'),
                            backgroundColor: AppTheme.deviceOnline,
                          ),
                        );
                      },
                      icon: const Icon(Icons.copy, size: 18),
                      label: const Text('Copy Plan Report', style: TextStyle(fontWeight: FontWeight.bold)),
                    ),
                  ),
                  const SizedBox(width: 12),
                  OutlinedButton.icon(
                    style: OutlinedButton.styleFrom(
                      padding: const EdgeInsets.symmetric(vertical: 14, horizontal: 16),
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
                    ),
                    onPressed: () {
                      Navigator.pop(ctx);
                      context.go('/aquadoc');
                    },
                    icon: const Icon(Icons.psychology, size: 18),
                    label: const Text('Ask AquaDoc'),
                  ),
                ],
              ),
              const SizedBox(height: 16),
            ],
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final farmUnits = ref.watch(farmUnitsProvider).units;
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
            icon: const Icon(Icons.assignment_outlined, color: Colors.cyanAccent),
            tooltip: 'Generate Farmer Production Plan Report',
            onPressed: () => _showFarmerPlanReportSheet(context),
          ),
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
          if (farmUnits.isNotEmpty) ...[
            Container(
              padding: const EdgeInsets.all(14),
              decoration: BoxDecoration(
                color: Theme.of(context).colorScheme.primary.withOpacity(0.12),
                borderRadius: BorderRadius.circular(14),
                border: Border.all(color: Theme.of(context).colorScheme.primary.withOpacity(0.3)),
              ),
              child: Row(
                children: [
                  Icon(Icons.waves, color: Theme.of(context).colorScheme.primary, size: 20),
                  const SizedBox(width: 10),
                  const Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text('Import Real Pond Configuration', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 13)),
                        Text('Load your exact pond dimensions & stock', style: TextStyle(fontSize: 11, color: Colors.grey)),
                      ],
                    ),
                  ),
                  PopupMenuButton<FarmUnit>(
                    icon: const Icon(Icons.file_download_outlined),
                    tooltip: 'Select Pond to Import',
                    onSelected: _importFromUnit,
                    itemBuilder: (ctx) => farmUnits
                        .map((u) => PopupMenuItem(
                              value: u,
                              child: Text('${u.name} (${u.fishCount} fish, ${u.lengthM}×${u.widthM}m)'),
                            ))
                        .toList(),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 14),
          ],

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
                if (_selectedPreset == 'custom') ...[
                  const SizedBox(width: 8),
                  _PresetChip(
                    label: '📌 Custom / Imported Pond',
                    isSelected: true,
                    onTap: () {},
                  ),
                ],
              ],
            ),
          ),
          // FARMER PLAN REPORT QUICK TRIGGER CARD
          InkWell(
            onTap: () => _showFarmerPlanReportSheet(context),
            borderRadius: BorderRadius.circular(16),
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
              decoration: BoxDecoration(
                gradient: const LinearGradient(
                  colors: [Color(0xFF0F2027), Color(0xFF203A43), Color(0xFF2C5364)],
                  begin: Alignment.topLeft,
                  end: Alignment.bottomRight,
                ),
                borderRadius: BorderRadius.circular(16),
                border: Border.all(color: Colors.cyanAccent.withValues(alpha: 0.5)),
                boxShadow: [
                  BoxShadow(
                    color: Colors.cyan.withValues(alpha: 0.15),
                    blurRadius: 10,
                    offset: const Offset(0, 4),
                  ),
                ],
              ),
              child: Row(
                children: [
                  Container(
                    padding: const EdgeInsets.all(8),
                    decoration: BoxDecoration(
                      color: Colors.cyanAccent.withValues(alpha: 0.15),
                      borderRadius: BorderRadius.circular(10),
                    ),
                    child: const Icon(Icons.assignment, color: Colors.cyanAccent, size: 22),
                  ),
                  const SizedBox(width: 12),
                  const Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          'Farmer Production Plan Report',
                          style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 13.5),
                        ),
                        SizedBox(height: 2),
                        Text(
                          'View 180-day feeding budget, density & harvest roadmap',
                          style: TextStyle(color: Colors.white70, fontSize: 11),
                        ),
                      ],
                    ),
                  ),
                  const Icon(Icons.chevron_right, color: Colors.cyanAccent, size: 20),
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),

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

class _ReportSectionCard extends StatelessWidget {
  final String title;
  final IconData icon;
  final Color iconColor;
  final List<_ReportRow> rows;

  const _ReportSectionCard({
    required this.title,
    required this.icon,
    required this.iconColor,
    required this.rows,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        color: Colors.white.withValues(alpha: 0.05),
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: Colors.white.withValues(alpha: 0.12)),
      ),
      padding: const EdgeInsets.all(14),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(icon, color: iconColor, size: 18),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  title,
                  style: TextStyle(
                    color: iconColor,
                    fontSize: 13,
                    fontWeight: FontWeight.bold,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 10),
          const Divider(height: 1, color: Colors.white12),
          const SizedBox(height: 8),
          ...rows,
        ],
      ),
    );
  }
}

class _ReportRow extends StatelessWidget {
  final String label;
  final String value;
  final bool isHighlight;

  const _ReportRow({
    required this.label,
    required this.value,
    this.isHighlight = false,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Expanded(
            flex: 5,
            child: Text(
              label,
              style: TextStyle(
                color: Colors.white70,
                fontSize: 11.5,
                fontWeight: isHighlight ? FontWeight.w600 : FontWeight.normal,
              ),
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            flex: 6,
            child: Text(
              value,
              textAlign: TextAlign.right,
              style: TextStyle(
                color: isHighlight ? Colors.cyanAccent : Colors.white,
                fontSize: 12,
                fontWeight: isHighlight ? FontWeight.bold : FontWeight.w500,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

