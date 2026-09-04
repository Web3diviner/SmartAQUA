import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/models/farm_unit.dart';
import '../../../../core/providers/farm_provider.dart';
import '../../../../core/theme/app_theme.dart';

class DiseaseTriageScreen extends ConsumerStatefulWidget {
  const DiseaseTriageScreen({super.key});

  @override
  ConsumerState<DiseaseTriageScreen> createState() => _DiseaseTriageScreenState();
}

class _DiseaseTriageScreenState extends ConsumerState<DiseaseTriageScreen> {
  final List<String> _selectedSymptoms = [
    'Skin ulcers, red lesions & hemorrhagic sores',
    'Loss of appetite / Feed refusal',
  ];

  String _species = 'Clarias gariepinus (African Catfish)';
  int _mortality24h = 14;
  int _durationDays = 2;
  double _waterTempC = 28.5;
  double _dissolvedOxygen = 4.8;
  double _ammoniaTan = 0.40;

  bool _isAssessed = true;

  final List<String> _allSymptoms = [
    'Surface piping / Gasping at water surface',
    'Loss of appetite / Feed refusal',
    'Skin ulcers, red lesions & hemorrhagic sores',
    'Broken head / Skull fissure & head swelling',
    'Saddleback lesion / Columnaris & fin rot',
    'Flashing, scratching against tank walls & excess mucus',
    'Abdominal distension (Dropsy) / Popeye',
    'Pale, congested, or necrotic brown gills',
    'Rotten egg odor / Dark bottom sludge (H2S toxicity)',
    'Lethargy & sluggish bottom resting',
  ];

  void _toggleSymptom(String s) {
    setState(() {
      if (_selectedSymptoms.contains(s)) {
        _selectedSymptoms.remove(s);
      } else {
        _selectedSymptoms.add(s);
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final hasUlcers = _selectedSymptoms.contains('Skin ulcers, red lesions & hemorrhagic sores');
    final hasPiping = _selectedSymptoms.contains('Surface piping / Gasping at water surface');
    final hasBrokenHead = _selectedSymptoms.contains('Broken head / Skull fissure & head swelling');
    final hasColumnaris = _selectedSymptoms.contains('Saddleback lesion / Columnaris & fin rot');
    final hasFlashing = _selectedSymptoms.contains('Flashing, scratching against tank walls & excess mucus');
    final hasDropsy = _selectedSymptoms.contains('Abdominal distension (Dropsy) / Popeye');

    String primaryDiagnosis = 'Motile Aeromonas Septicemia (MAS / Red Sore Disease)';
    double probability = 0.92;
    String causativeAgent = 'Aeromonas hydrophila (Bacterial Opportunist)';
    String severity = 'CRITICAL BIOSECURITY RISK';
    Color severityColor = Colors.red;

    if (hasBrokenHead) {
      primaryDiagnosis = 'Broken Head Disease (Nutritional / Bacterial)';
      probability = 0.95;
      causativeAgent = 'Ascorbic acid deficiency compounded by Edwardsiella ictaluri';
      severity = 'HIGH SEVERITY';
      severityColor = Colors.orange[800]!;
    } else if (hasColumnaris) {
      primaryDiagnosis = 'Columnaris Disease (Saddleback Syndrome)';
      probability = 0.89;
      causativeAgent = 'Flavobacterium columnare (Gram-negative Rod)';
      severity = 'HIGH CONTAGION';
      severityColor = Colors.orange[800]!;
    } else if (hasPiping && !hasUlcers) {
      primaryDiagnosis = 'Severe Acute Hypoxia & Nitrite Asphyxiation';
      probability = 0.94;
      causativeAgent = 'Environmental Oxygen Depletion / Methemoglobinemia';
      severity = 'IMMEDIATE EMERGENCY';
      severityColor = Colors.red;
    } else if (hasFlashing) {
      primaryDiagnosis = 'Ectoparasitic Trichodiniasis / Gyrodactylus Infestation';
      probability = 0.87;
      causativeAgent = 'Trichodina heterodentata / Dactylogyrus flukes';
      severity = 'MODERATE INFESTATION';
      severityColor = Colors.amber[900]!;
    } else if (hasDropsy) {
      primaryDiagnosis = 'Infectious Dropsy & Abdominal Ascites';
      probability = 0.86;
      causativeAgent = 'Pseudomonas / Aeromonas co-infection & renal failure';
      severity = 'HIGH BIOSECURITY RISK';
      severityColor = Colors.red;
    }

    return Scaffold(
      appBar: AppBar(
        title: const Row(
          children: [
            Icon(Icons.medical_services, color: Colors.redAccent),
            SizedBox(width: 8),
            Text('Clinical Disease Triage'),
          ],
        ),
        actions: [
          IconButton(
            icon: const Icon(Icons.psychology),
            tooltip: 'Consult AquaDoc AI',
            onPressed: () => context.go('/aquadoc'),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          // Header Card
          Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              gradient: LinearGradient(
                colors: [
                  Colors.blueGrey[900]!,
                  Colors.blueGrey[800]!,
                ],
              ),
              borderRadius: BorderRadius.circular(16),
            ),
            child: Row(
              children: [
                Container(
                  padding: const EdgeInsets.all(10),
                  decoration: BoxDecoration(
                    color: Colors.red.withOpacity(0.2),
                    shape: BoxShape.circle,
                  ),
                  child: const Icon(Icons.healing, color: Colors.redAccent, size: 28),
                ),
                const SizedBox(width: 14),
                const Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'AquaDoc Pathology & Triage Engine',
                        style: TextStyle(color: Colors.white, fontSize: 16, fontWeight: FontWeight.bold),
                      ),
                      SizedBox(height: 2),
                      Text(
                        'Grounded in peer-reviewed tropical veterinary literature',
                        style: TextStyle(color: Colors.white70, fontSize: 12),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 14),

          // Load from My Ponds Selector
          Builder(builder: (context) {
            final farmUnits = ref.watch(farmUnitsProvider).units;
            if (farmUnits.isEmpty) return const SizedBox.shrink();

            return Container(
              margin: const EdgeInsets.only(bottom: 16),
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: Colors.cyan.withOpacity(0.1),
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: Colors.cyan.withOpacity(0.3)),
              ),
              child: Row(
                children: [
                  const Icon(Icons.waves, color: Colors.cyanAccent, size: 18),
                  const SizedBox(width: 10),
                  const Expanded(
                    child: Text(
                      'Load Pond Water & Stock Profile:',
                      style: TextStyle(fontSize: 12, fontWeight: FontWeight.bold),
                    ),
                  ),
                  PopupMenuButton<FarmUnit>(
                    child: Container(
                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                      decoration: BoxDecoration(
                        color: Colors.cyan.withOpacity(0.2),
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: const Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Text('Select Pond', style: TextStyle(fontSize: 12, color: Colors.cyanAccent, fontWeight: FontWeight.bold)),
                          Icon(Icons.arrow_drop_down, color: Colors.cyanAccent, size: 18),
                        ],
                      ),
                    ),
                    onSelected: (unit) {
                      setState(() {
                        _species = unit.species;
                        if (unit.manualTemp != null) _waterTempC = unit.manualTemp!;
                        if (unit.manualDO != null) _dissolvedOxygen = unit.manualDO!;
                        if (unit.manualTAN != null) _ammoniaTan = unit.manualTAN!;
                      });
                      ScaffoldMessenger.of(context).showSnackBar(
                        SnackBar(
                          content: Text('Loaded ${unit.name} telemetry & species profile!'),
                          backgroundColor: AppTheme.deviceOnline,
                        ),
                      );
                    },
                    itemBuilder: (ctx) => farmUnits
                        .map((u) => PopupMenuItem(
                              value: u,
                              child: Text('${u.name} (${u.species.split(" ").first})'),
                            ))
                        .toList(),
                  ),
                ],
              ),
            );
          }),

          // Symptoms Multi-Select Checklist
          Text(
            'Observed Clinical Symptoms (Tap to Select)',
            style: Theme.of(context).textTheme.titleSmall?.copyWith(fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: _allSymptoms.map((symptom) {
              final isSelected = _selectedSymptoms.contains(symptom);
              return FilterChip(
                label: Text(
                  symptom,
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: isSelected ? FontWeight.bold : FontWeight.normal,
                    color: isSelected ? Colors.red[900] : null,
                  ),
                ),
                selected: isSelected,
                selectedColor: Colors.red.withOpacity(0.18),
                checkmarkColor: Colors.red[900],
                onSelected: (_) => _toggleSymptom(symptom),
              );
            }).toList(),
          ),
          const SizedBox(height: 20),

          // Morbidity & Mortality Inputs
          Row(
            children: [
              Expanded(
                child: Card(
                  elevation: 1,
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                  child: Padding(
                    padding: const EdgeInsets.all(12),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const Text('24h Mortality', style: TextStyle(fontSize: 11, color: Colors.grey)),
                        const SizedBox(height: 4),
                        Row(
                          children: [
                            IconButton(
                              icon: const Icon(Icons.remove_circle_outline, size: 20),
                              onPressed: () {
                                if (_mortality24h > 0) setState(() => _mortality24h--);
                              },
                            ),
                            Text('$_mortality24h fish', style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 15)),
                            IconButton(
                              icon: const Icon(Icons.add_circle_outline, size: 20),
                              onPressed: () => setState(() => _mortality24h++),
                            ),
                          ],
                        ),
                      ],
                    ),
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Card(
                  elevation: 1,
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                  child: Padding(
                    padding: const EdgeInsets.all(12),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const Text('Symptom Duration', style: TextStyle(fontSize: 11, color: Colors.grey)),
                        const SizedBox(height: 4),
                        Row(
                          children: [
                            IconButton(
                              icon: const Icon(Icons.remove_circle_outline, size: 20),
                              onPressed: () {
                                if (_durationDays > 1) setState(() => _durationDays--);
                              },
                            ),
                            Text('$_durationDays days', style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 15)),
                            IconButton(
                              icon: const Icon(Icons.add_circle_outline, size: 20),
                              onPressed: () => setState(() => _durationDays++),
                            ),
                          ],
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 24),

          // Differential Diagnosis Result Card
          Card(
            elevation: 3,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(16),
              side: BorderSide(color: severityColor.withOpacity(0.5), width: 1.5),
            ),
            child: Padding(
              padding: const EdgeInsets.all(18),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                        decoration: BoxDecoration(
                          color: severityColor.withOpacity(0.15),
                          borderRadius: BorderRadius.circular(10),
                        ),
                        child: Text(
                          severity,
                          style: TextStyle(color: severityColor, fontSize: 11, fontWeight: FontWeight.bold),
                        ),
                      ),
                      Text(
                        'Confidence: ${(probability * 100).toInt()}%',
                        style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 13, color: Colors.blueAccent),
                      ),
                    ],
                  ),
                  const SizedBox(height: 12),
                  Text(
                    primaryDiagnosis,
                    style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    'Etiology: $causativeAgent',
                    style: TextStyle(fontSize: 12, color: Colors.grey[600], fontStyle: FontStyle.italic),
                  ),
                  const SizedBox(height: 16),
                  const Divider(height: 1),
                  const SizedBox(height: 14),

                  // Treatment Protocol Step by Step
                  Text(
                    'Step-by-Step Clinical Treatment Protocol',
                    style: Theme.of(context).textTheme.titleSmall?.copyWith(fontWeight: FontWeight.bold),
                  ),
                  const SizedBox(height: 10),
                  _TreatmentStep(
                    stepNum: '1',
                    title: 'Immediate Water Sanitation & 50% Exchange',
                    desc: 'Drain 50% of tank volume from bottom drain to remove pathogenic sludge. Refill with pre-aerated water.',
                  ),
                  const SizedBox(height: 8),
                  _TreatmentStep(
                    stepNum: '2',
                    title: 'Non-Iodized Salinity Bath (3 - 5 g/L)',
                    desc: 'Apply coarse raw salt to reduce osmoregulatory stress, inhibit bacterial proliferation, and reduce nitrite toxicity.',
                  ),
                  const SizedBox(height: 8),
                  _TreatmentStep(
                    stepNum: '3',
                    title: 'Ration Reduction & Medicated Feed Protocol',
                    desc: 'Cut feed ration by 50%. Administer Oxytetracycline (50-75 mg/kg fish biomass/day) incorporated in feed for 7 consecutive days.',
                  ),
                  const SizedBox(height: 8),
                  _TreatmentStep(
                    stepNum: '4',
                    title: 'Withdrawal Period & Biosecurity Interlock',
                    desc: 'Enforce strict 21-day withdrawal period before any harvest. Disinfect nets and sampling equipment in 100ppm chlorine.',
                  ),
                ],
              ),
            ),
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
                        content: Text('Triage case report injected into AquaDoc!'),
                        backgroundColor: AppTheme.deviceOnline,
                      ),
                    );
                    context.go('/aquadoc');
                  },
                  icon: const Icon(Icons.psychology),
                  label: const Text('Consult AquaDoc AI'),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: () => context.go('/twin'),
                  icon: const Icon(Icons.hub),
                  label: const Text('View AquaTwin'),
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

class _TreatmentStep extends StatelessWidget {
  final String stepNum;
  final String title;
  final String desc;

  const _TreatmentStep({
    required this.stepNum,
    required this.title,
    required this.desc,
  });

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
          width: 22,
          height: 22,
          decoration: BoxDecoration(
            color: Theme.of(context).colorScheme.primary.withOpacity(0.15),
            shape: BoxShape.circle,
          ),
          alignment: Alignment.center,
          child: Text(
            stepNum,
            style: TextStyle(
              fontSize: 11,
              fontWeight: FontWeight.bold,
              color: Theme.of(context).colorScheme.primary,
            ),
          ),
        ),
        const SizedBox(width: 10),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(title, style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 13)),
              const SizedBox(height: 2),
              Text(desc, style: TextStyle(fontSize: 12, color: Colors.grey[600], height: 1.3)),
            ],
          ),
        ),
      ],
    );
  }
}
