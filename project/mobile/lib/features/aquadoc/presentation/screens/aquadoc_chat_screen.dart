import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../core/providers/device_provider.dart';
import '../../../../core/providers/farm_provider.dart';
import '../../../../core/providers/monitoring_provider.dart';

class AquaDocMessage {
  final String id;
  final String text;
  final bool isUser;
  final DateTime timestamp;
  final String severity; // info, warning, critical
  final List<String> missingData;

  const AquaDocMessage({
    required this.id,
    required this.text,
    required this.isUser,
    required this.timestamp,
    this.severity = 'info',
    this.missingData = const [],
  });
}

class AquaDocChatScreen extends ConsumerStatefulWidget {
  const AquaDocChatScreen({super.key});

  @override
  ConsumerState<AquaDocChatScreen> createState() => _AquaDocChatScreenState();
}

class _AquaDocChatScreenState extends ConsumerState<AquaDocChatScreen> {
  final TextEditingController _textController = TextEditingController();
  final ScrollController _scrollController = ScrollController();
  bool _isTyping = false;

  final List<AquaDocMessage> _messages = [
    AquaDocMessage(
      id: 'msg-01',
      text: 'Hello! I am AquaDoc, your aquaculture clinical advisor.\n\n'
          'I am connected to your live farm telemetry. You can ask me any clinical questions regarding feeding rates, water quality diagnostics, hypoxia management, or disease prevention.\n\n'
          'How can I assist your farm today?',
      isUser: false,
      timestamp: DateTime.now().subtract(const Duration(minutes: 5)),
      severity: 'info',
    ),
  ];

  final List<String> _quickPrompts = [
    'Red sores & skin ulcers (MAS)',
    'Broken head & skull swelling',
    'Columnaris & fin rot treatment',
    'White spots / Ich & scratching',
    'Surface piping / DO emergency',
    'Swollen belly (Dropsy) protocol',
    'Fish refusing feed / loss of appetite',
    'Medication & salt dosage calculator',
  ];

  @override
  void dispose() {
    _textController.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  void _sendMessage(String text) {
    if (text.trim().isEmpty) return;

    final userMsg = AquaDocMessage(
      id: 'usr-${DateTime.now().millisecondsSinceEpoch}',
      text: text.trim(),
      isUser: true,
      timestamp: DateTime.now(),
    );

    setState(() {
      _messages.add(userMsg);
      _isTyping = true;
    });
    _textController.clear();
    _scrollToBottom();

    // Generate grounded clinical AI response
    Future.delayed(const Duration(milliseconds: 1000), () {
      if (!mounted) return;
      final aiResponse = _generateAiResponse(text);
      setState(() {
        _messages.add(aiResponse);
        _isTyping = false;
      });
      _scrollToBottom();
    });
  }

  AquaDocMessage _generateAiResponse(String prompt) {
    final lower = prompt.toLowerCase();
    final now = DateTime.now();
    final farmUnits = ref.read(farmUnitsProvider).units;
    final activeUnit = farmUnits.firstOrNull;
    final sensorData = ref.read(sensorDataProvider).currentData;
    
    final unitName = activeUnit?.name ?? 'Primary Culture Unit';
    final species = activeUnit?.species ?? 'Clarias gariepinus (African Catfish)';
    final tempStr = (sensorData != null && sensorData.waterTemperature > 0)
        ? '${sensorData.waterTemperature.toStringAsFixed(1)}°C'
        : (activeUnit?.manualTemp != null ? '${activeUnit!.manualTemp!.toStringAsFixed(1)}°C' : '28.0°C');

    // 1. Red sores, Ulcers, Hemorrhage, MAS (Aeromonas hydrophila)
    if (lower.contains('ulcer') ||
        lower.contains('sore') ||
        lower.contains('red spot') ||
        lower.contains('hemorrhag') ||
        lower.contains('aeromonas') ||
        lower.contains('mas') ||
        lower.contains('red disease') ||
        (lower.contains('red') && (lower.contains('skin') || lower.contains('belly') || lower.contains('body')))) {
      return AquaDocMessage(
        id: 'ai-${now.millisecondsSinceEpoch}',
        text: '🏥 Clinical Diagnosis: Motile Aeromonas Septicemia (MAS / Red Sore Disease)\n\n'
            'Target Subject: $species in $unitName (Current Temp: $tempStr)\n'
            'Primary Pathogen: Aeromonas hydrophila / Aeromonas sobria (Gram-negative opportunistic bacterium)\n'
            'Severity: HIGH BIOSECURITY & PATHOLOGY RISK\n\n'
            '📋 Clinical Findings:\n'
            '• Cutaneous ulcerations, dermal necrosis, petechial hemorrhages along the ventral belly, fin bases, and perianal region.\n'
            '• Triggered by organic bottom sludge accumulation, handling abrasions, or sudden temperature/pH fluctuations.\n\n'
            '💉 Immediate Emergency Protocol:\n'
            '1. Water Exchange: Perform an immediate 30% to 50% water exchange with clean, dechlorinated, well-aerated water to drop bacterial colony forming units (CFU/mL).\n'
            '2. Feeding Pause: Halt regular commercial feed for 24 hours. Digestion in septicemic fish accelerates metabolic acidosis.\n'
            '3. Salt Immersion: Administer non-iodized salt (NaCl) at 3.0 to 5.0 g/L (3–5 ppt) in static tanks or a short-duration dip at 10.0–15.0 g/L (10–15 ppt) for 10–15 minutes under continuous aeration.\n'
            '4. Antibiotic Therapy (If systemic mortality > 1%/day):\n'
            '   • Oxytetracycline (OTC): 50 to 75 mg per kg of fish biomass daily, incorporated into feed using 5 mL vegetable oil binder per kg feed, for 7 to 10 consecutive days.\n'
            '   • Alternative: Florfenicol at 10 mg/kg fish body weight daily in feed for 10 days.\n'
            '   • Warning: Observe a strict 21-day withdrawal period before harvest.\n\n'
            '🛡️ Preventive Biosecurity:\n'
            '• Siphon bottom anaerobic sediment and maintain dissolved oxygen > 4.5 mg/L.',
        isUser: false,
        timestamp: now,
        severity: 'critical',
        missingData: const ['Water Hardness', 'Nitrite NO2'],
      );
    }

    // 2. Broken Head Disease / Skull Fissure / Swollen Skull (Edwardsiella / Ascorbic Acid deficiency)
    if (lower.contains('broken head') ||
        lower.contains('skull') ||
        lower.contains('head crack') ||
        lower.contains('crack on head') ||
        lower.contains('head swell') ||
        lower.contains('edwardsiella')) {
      return AquaDocMessage(
        id: 'ai-${now.millisecondsSinceEpoch}',
        text: '🏥 Clinical Diagnosis: Broken Head Disease (Edwardsiellosis & Nutritional Ascorbic Acid Deficiency)\n\n'
            'Target Subject: $species in $unitName\n'
            'Primary Etiology: Chronic dietary Vitamin C (Ascorbic Acid) deficiency complicated by secondary Edwardsiella ictaluri / Edwardsiella tarda bacterial infection.\n'
            'Severity: CRITICAL SKELETAL & PATHOLOGICAL CONDITION\n\n'
            '📋 Pathological Signs:\n'
            '• Transverse skull fissure (crack across the parietal-occipital cranial suture).\n'
            '• Cranial inflammation, lateral curvature of spine (scoliosis/lordosis), and erratic spinning behavior.\n\n'
            '💉 Therapeutic Protocol:\n'
            '1. Vitamin C Megadose: Immediately switch to or top-dress feed with heat-stable L-Ascorbyl-2-Polyphosphate (Vitamin C) at 500 to 1,000 mg per kg of feed for 14 to 21 consecutive days.\n'
            '2. Antimicrobial Therapy: Top-dress feed with Oxytetracycline @ 75 mg/kg fish biomass/day for 10 days to arrest Edwardsiella cranial bone osteomyelitis.\n'
            '3. Water Treatment: Siphon anaerobic bottom muck immediately. Edwardsiella reservoirs multiply rapidly in hydrogen-sulfide rich sludge.',
        isUser: false,
        timestamp: now,
        severity: 'critical',
        missingData: const ['Feed Manufacturing Date', 'Feed Storage Temp'],
      );
    }

    // 3. Columnaris / Saddleback / Fin Rot / Tail Rot / Cotton Mouth (Flavobacterium columnare)
    if (lower.contains('columnaris') ||
        lower.contains('saddleback') ||
        lower.contains('fin rot') ||
        lower.contains('tail rot') ||
        lower.contains('mouth rot') ||
        lower.contains('cotton mouth') ||
        lower.contains('frayed fin')) {
      return AquaDocMessage(
        id: 'ai-${now.millisecondsSinceEpoch}',
        text: '🏥 Clinical Diagnosis: Columnaris Disease (Saddleback Syndrome & Necrotic Fin Rot)\n\n'
            'Target Subject: $species in $unitName (Water Temp: $tempStr)\n'
            'Primary Pathogen: Flavobacterium columnare (Gram-negative gliding flexibacterium)\n'
            'Severity: HIGH CONTAGION RISK\n\n'
            '📋 Clinical Characteristics:\n'
            '• Characteristic white/yellowish "saddleback" discoloration band across the dorsal fin region.\n'
            '• Frayed, necrotic fins with white margins; eroded mouth tissues mimicking fungal growth ("cotton mouth").\n'
            '• Virulence spikes dramatically when water temperatures exceed 27°C.\n\n'
            '💉 Treatment Protocol:\n'
            '1. Chemical Bath / Tank Treatment:\n'
            '   • Potassium Permanganate (KMnO4): 2.0 to 4.0 mg/L (ppm) prolonged bath for 30–60 minutes in aerated quarantine or treatment tank.\n'
            '   • Alternatively: Salt bath at 5.0 to 10.0 g/L (5–10 ppt) for 20 minutes.\n'
            '2. Medicated Feed (Internal Infection):\n'
            '   • Oxytetracycline @ 50 mg/kg fish weight/day in feed for 10 days.\n'
            '3. Thermal & Husbandry Care: Reduce feeding during peak afternoon temperatures to decrease metabolic gill stress.',
        isUser: false,
        timestamp: now,
        severity: 'warning',
        missingData: const ['Organic Load (TSS)', 'Alkalinity'],
      );
    }

    // 4. White Spot / Ich / Ectoparasites / Scratching / Flashing / Trichodina / Flukes
    if (lower.contains('white spot') ||
        lower.contains('ich') ||
        lower.contains('scratching') ||
        lower.contains('flashing') ||
        lower.contains('rubbing') ||
        lower.contains('trichodina') ||
        lower.contains('parasite') ||
        lower.contains('fluke') ||
        lower.contains('gyrodactylus') ||
        lower.contains('dactylogyrus') ||
        lower.contains('excess mucus')) {
      return AquaDocMessage(
        id: 'ai-${now.millisecondsSinceEpoch}',
        text: '🏥 Clinical Diagnosis: Ectoparasitic Infestation (Ichthyophthiriasis / Trichodiniasis / Monogenean Flukes)\n\n'
            'Target Subject: $species in $unitName\n'
            'Primary Parasites: Ichthyophthirius multifiliis (White Spot ciliate), Trichodina spp., or Gyrodactylus / Dactylogyrus skin & gill flukes.\n'
            'Severity: MODERATE TO HIGH CONTAGION\n\n'
            '📋 Clinical Signs:\n'
            '• Punctate white salt-like nodules on skin/fins, erratic "flashing" (darting and scraping against walls), excessive cloudy epidermal slime, rapid opercular breathing.\n\n'
            '💉 Treatment & Eradication Protocol:\n'
            '1. Thermal Acceleration (For White Spot Ich):\n'
            '   • If in controlled indoor/tanks, safely elevate temperature to 30°C–32°C for 4–5 days to accelerate the parasite life cycle from encysted tomont to vulnerable free-swimming theronts.\n'
            '2. Formalin Treatment:\n'
            '   • 37% Formaldehyde (Formalin) at 15 to 25 ppm (15–25 mL per 1,000 Liters) with maximum mechanical aeration.\n'
            '   • ⚠️ Caution: Never use Formalin if dissolved oxygen is below 5.0 mg/L or in water with heavy phytoplankton blooms.\n'
            '3. Prolonged Salt Bath:\n'
            '   • Uniodized Salt (NaCl) at 3.0 to 5.0 g/L (3–5 ppt) for 5 to 7 continuous days.\n'
            '4. For Flukes (Gyrodactylus):\n'
            '   • Praziquantel at 2.0 to 3.0 mg/L (ppm) bath for 24 hours.',
        isUser: false,
        timestamp: now,
        severity: 'warning',
        missingData: const ['Dissolved Oxygen Level', 'Pond Volume (Liters)'],
      );
    }

    // 5. Dropsy / Swollen Belly / Bloated Abdomen / Popeye (Exophthalmia)
    if (lower.contains('dropsy') ||
        lower.contains('swollen belly') ||
        lower.contains('bloat') ||
        lower.contains('big belly') ||
        lower.contains('ascites') ||
        lower.contains('popeye') ||
        lower.contains('bulging eye') ||
        lower.contains('exophthalm')) {
      return AquaDocMessage(
        id: 'ai-${now.millisecondsSinceEpoch}',
        text: '🏥 Clinical Diagnosis: Infectious Dropsy & Abdominal Ascites with Exophthalmia (Popeye)\n\n'
            'Target Subject: $species in $unitName\n'
            'Etiology: Systemic bacterial septicemia (Pseudomonas / Aeromonas co-infection) leading to acute renal and osmoregulatory failure, causing fluid accumulation in the peritoneal cavity.\n'
            'Severity: CRITICAL MORTALITY RISK\n\n'
            '📋 Clinical Features:\n'
            '• Severe coelomic distension with transparent/yellow ascitic fluid, protrusion of eyeballs (exophthalmia), and lethargic bottom resting.\n\n'
            '💉 Clinical Treatment:\n'
            '1. Osmotic Pressure Relief: Immediately transfer affected fish to a 3.0 to 5.0 ppt (3–5 g/L) salt bath to ease kidney osmotic workload.\n'
            '2. Fasting: Fast the population for 48 hours to stop gastrointestinal gas fermentation.\n'
            '3. Quarantine: Remove severely distended fish immediately to prevent cannibalism from spreading high bacterial loads.\n'
            '4. Antibiotic Intervention for Remaining Cohort:\n'
            '   • Medicated ration with Oxytetracycline @ 75 mg/kg fish weight/day or Florfenicol @ 10 mg/kg/day for 10 days.',
        isUser: false,
        timestamp: now,
        severity: 'critical',
        missingData: const ['Total Feed Rations (kg/day)', 'Daily Water Exchange %'],
      );
    }

    // 6. Fungal Infection / Saprolegnia / Cotton wool / White fluff
    if (lower.contains('fung') ||
        lower.contains('cotton') ||
        lower.contains('fluff') ||
        lower.contains('wool') ||
        lower.contains('saprolegnia') ||
        lower.contains('white fuzz')) {
      return AquaDocMessage(
        id: 'ai-${now.millisecondsSinceEpoch}',
        text: '🏥 Clinical Diagnosis: Saprolegniasis (Water Mold / Cotton Wool Disease)\n\n'
            'Target Subject: $species in $unitName\n'
            'Primary Pathogen: Saprolegnia parasitica (Oomycete fungus)\n'
            'Severity: MODERATE TO HIGH SECONDARY INFECTION\n\n'
            '📋 Pathology:\n'
            '• White or grayish-brown fluffy tufts resembling cotton wool on skin, fins, or gills. Saprolegnia is an obligate opportunist that only colonizes pre-existing lesions caused by rough netting, ectoparasite bites, or temperature shock (<22°C).\n\n'
            '💉 Therapeutic Protocol:\n'
            '1. Salt Immersion Therapy:\n'
            '   • Salt dip at 15.0 to 20.0 g/L (1.5–2.0%) for 10 to 15 minutes with strong aeration.\n'
            '   • Or prolonged tank bath at 3.0 to 5.0 g/L (3–5 ppt) for 7 days.\n'
            '2. Potassium Permanganate Bath:\n'
            '   • 3.0 to 4.0 mg/L (ppm) KMnO4 for 45 minutes.\n'
            '3. Preventive Care: Eliminate sharp tank surfaces, avoid coarse handling nets, and investigate underlying primary ectoparasites (Trichodina/Ich).',
        isUser: false,
        timestamp: now,
        severity: 'warning',
        missingData: const ['Water Temperature Trend', 'Handling History'],
      );
    }

    // 7. Surface Piping / Gasping / Floating at Surface / Hypoxia Emergency
    if (lower.contains('piping') ||
        lower.contains('gasp') ||
        lower.contains('surface') ||
        lower.contains('gulping') ||
        lower.contains('floating') ||
        lower.contains('suffocat') ||
        lower.contains('oxygen') ||
        lower.contains('do') ||
        lower.contains('hypoxia')) {
      return AquaDocMessage(
        id: 'ai-${now.millisecondsSinceEpoch}',
        text: '🚨 EMERGENCY CLINICAL ALERT: Acute Hypoxia & Respiratory Asphyxiation\n\n'
            'Target Subject: $species in $unitName (Current Temp: $tempStr)\n'
            'Condition: Critical Dissolved Oxygen (DO) Depletion (< 3.0 mg/L) or Nitrite-induced Methemoglobinemia.\n'
            'Severity: IMMEDIATE FATAL RISK IF UNRESOLVED\n\n'
            '📋 Clinical Urgency:\n'
            '• Fish congregating at the surface, gulping atmospheric air, lethargy, flared opercula, and loss of equilibrium.\n\n'
            '⚡ Immediate Emergency Protocol:\n'
            '1. Mechanical Aeration: Immediately engage all available air pumps, venturi injectors, and paddlewheel aerators at 100% capacity.\n'
            '2. EMERGENCY FEEDING HALT: Cease all automated and manual feeding instantly! Digesting a single feed ration increases fish oxygen demand by 40% to 60% (Specific Dynamic Action).\n'
            '3. Freshwater Flush: Turn on fresh, highly oxygenated incoming water supply.\n'
            '4. Nitrite Poisoning Check: If oxygen levels are high but fish still pipe, the cause is Nitrite (NO2-) toxicity (Brown Blood Disease). Dose non-iodized salt (NaCl) at 1.0 to 2.0 kg per 1,000 Liters (1–2 ppt) immediately. Chloride ions (Cl-) competitively block toxic nitrite absorption across gill membranes.',
        isUser: false,
        timestamp: now,
        severity: 'critical',
        missingData: const ['Live DO (mg/L)', 'Nitrite NO2 (mg/L)'],
      );
    }

    // 8. Ammonia Toxicity / TAN / NH3 Poisoning / pH Shock
    if (lower.contains('tan') ||
        lower.contains('ammonia') ||
        lower.contains('nh3') ||
        lower.contains('ph shock') ||
        lower.contains('acid') ||
        lower.contains('alkali')) {
      return AquaDocMessage(
        id: 'ai-${now.millisecondsSinceEpoch}',
        text: '🏥 Clinical Diagnosis: Total Ammonia Nitrogen (TAN) & Toxic Free Ammonia (NH3) Toxicity\n\n'
            'Target Subject: $species in $unitName (Current Temp: $tempStr)\n'
            'Severity: CRITICAL TOXICOLOGICAL EMERGENCY\n\n'
            '📋 Toxic Mechanism:\n'
            '• Un-ionized ammonia (NH3) is 100x more toxic than ammonium (NH4+). Toxicity increases exponentially as water pH (> 8.0) and temperature (> 28°C) rise.\n'
            '• Safe threshold: Un-ionized NH3 must remain below 0.05 mg/L.\n\n'
            '💉 Immediate Corrective Actions:\n'
            '1. Emergency 50% Water Exchange: Pump out bottom water and introduce clean, fresh water immediately.\n'
            '2. Feed Suspension: Stop feeding for 48 hours. 80% of ammonia in aquaculture originates from digested feed protein excretion across gills.\n'
            '3. pH Moderation: Ensure pH does not exceed 8.0. If pH is high, water exchange or gentle aeration will lower free NH3 proportion.\n'
            '4. Siphon Waste: Remove uneaten feed pellets and feces from tank bottoms.',
        isUser: false,
        timestamp: now,
        severity: 'critical',
        missingData: const ['Water pH', 'Nitrite NO2'],
      );
    }

    // 9. Loss of Appetite / Feed Refusal / Slow Feeding
    if (lower.contains('appetite') ||
        lower.contains('refus') ||
        lower.contains('not eating') ||
        lower.contains('stop eating') ||
        lower.contains('slow feeding') ||
        lower.contains('feed intake')) {
      return AquaDocMessage(
        id: 'ai-${now.millisecondsSinceEpoch}',
        text: '🏥 Clinical Consultation: Feed Refusal & Anorexia Investigation Protocol\n\n'
            'Target Subject: $species in $unitName (Water Temp: $tempStr)\n'
            'Severity: EARLY WARNING SIGN\n\n'
            '📋 Step-by-Step Diagnostic Hierarchy:\n'
            '1. Dissolved Oxygen Check: If DO drops below 3.5 mg/L, fish immediately suppress appetite. Ensure DO is > 5.0 mg/L.\n'
            '2. Water Quality Spikes: Test Ammonia (TAN > 1.0 mg/L) and Nitrite (NO2 > 0.5 mg/L). Sub-lethal toxicities trigger immediate feed cessation.\n'
            '3. Feed Integrity & Mycotoxins: Inspect feed bags for moisture, mold, or rancid odor. Aflatoxin B1 contamination in humid storage causes acute liver damage and complete feed refusal.\n'
            '4. Temperature Shock: Any sudden temperature shift > 2.5°C over 6 hours induces metabolic shock.\n'
            '5. Incipient Pathogen: Inspect fish for subtle flashing, clamped fins, or red perianal coloration indicating early septicemia.\n\n'
            '💡 Action: Fast for 1 feeding cycle, test water parameters, and test feed with a small handful before resuming full rations.',
        isUser: false,
        timestamp: now,
        severity: 'warning',
        missingData: const ['Feed Batch Code', 'Ammonia TAN', 'DO (mg/L)'],
      );
    }

    // 10. Mortality / Fish Dying / Sudden Death / Post-Rain Disaster
    if (lower.contains('dying') ||
        lower.contains('dead') ||
        lower.contains('mortality') ||
        lower.contains('death') ||
        lower.contains('killed') ||
        lower.contains('rain')) {
      return AquaDocMessage(
        id: 'ai-${now.millisecondsSinceEpoch}',
        text: '🚨 URGENT MORTALITY PROTOCOL: Rapid Triage & Catastrophe Containment\n\n'
            'Target Subject: $species in $unitName (Water Temp: $tempStr)\n'
            'Severity: ACUTE EPIDEMIOLOGICAL EMERGENCY\n\n'
            '📋 Post-Mortem Investigation & Action Steps:\n'
            '1. Post-Rain Pond Inversion (Turnover):\n'
            '   • Cold rainwater cools the top surface layer, causing dense cold water to plunge and displace toxic anaerobic bottom sludge (H2S, Ammonia, zero DO).\n'
            '   • Action: Immediately run surface aerators, drain bottom sludge water, and broadcast agricultural lime (CaCO3) at 200–300 kg/ha (earthen) or 20–30 g/m³ (concrete) to neutralize acidity.\n'
            '2. Mortality Disposal (Biosecurity Rule #1):\n'
            '   • Net out all mortalities every 2 hours and deep-bury with lime away from the water table. Never leave dead fish in the water—cannibalism accelerates bacterial spread tenfold.\n'
            '3. Perform Immediate 50% Water Flush.\n'
            '4. Quarantine & Triage: Isolate moribund fish and inspect gills and skin for lesions or parasites.',
        isUser: false,
        timestamp: now,
        severity: 'critical',
        missingData: const ['24h Mortality Count', 'Total Biomass (kg)', 'DO Level'],
      );
    }

    // 11. Chemical / Medication / Salt Dosage Calculator
    if (lower.contains('dose') ||
        lower.contains('dosage') ||
        lower.contains('calculate') ||
        lower.contains('how much salt') ||
        lower.contains('how to treat') ||
        lower.contains('medicine') ||
        lower.contains('medication') ||
        lower.contains('antibiotic') ||
        lower.contains('oxytetracycline') ||
        lower.contains('potassium permanganate') ||
        lower.contains('formalin')) {
      return AquaDocMessage(
        id: 'ai-${now.millisecondsSinceEpoch}',
        text: '💊 Clinical Veterinary Pharmaceutical & Chemical Dosage Formulary\n\n'
            'Standardized Therapeutic Dosages for Aquaculture:\n\n'
            '1. Non-Iodized Salt (NaCl):\n'
            '   • Anti-Stress / Transport / Osmoregulation: 1.0 to 3.0 g/L (1–3 ppt / 1–3 kg per 1,000L).\n'
            '   • Prolonged Parasite & Bacterial Treatment: 3.0 to 5.0 g/L (3–5 ppt) for 5–7 days.\n'
            '   • Short-Duration Ectoparasite Dip: 10.0 to 15.0 g/L (10–15 ppt) for 10 to 15 minutes.\n\n'
            '2. Potassium Permanganate (KMnO4):\n'
            '   • External Disinfection & Columnaris Bath: 2.0 to 4.0 mg/L (ppm / 2–4 g per 1,000L). Maintain wine-red/pink color for 30–60 minutes. Neutralize with hydrogen peroxide if needed.\n\n'
            '3. Oxytetracycline HCl (OTC Medicated Feed):\n'
            '   • 50 to 75 mg pure active ingredient per kg total fish biomass daily in feed for 7 to 10 days.\n'
            '   • Preparation: Dissolve in small warm water, mix with 5 mL vegetable oil binder, coat feed, and air dry before feeding.\n\n'
            '4. Florfenicol (Aquaculture Antibiotic):\n'
            '   • 10 mg per kg fish biomass daily in feed for 10 days.\n\n'
            '5. Formalin (37% Formaldehyde):\n'
            '   • 15 to 25 ppm (15–25 mL per 1,000L) with continuous heavy aeration.\n\n'
            '⚠️ Crucial Safety Note: Always calculate dosages based on exact water volume (L = Length x Width x Depth x 1000) or total fish weight in kg.',
        isUser: false,
        timestamp: now,
        severity: 'info',
      );
    }

    // 12. Cannibalism, Grading & Size Heterogeneity
    if (lower.contains('cannibal') ||
        lower.contains('eating each other') ||
        lower.contains('biting') ||
        lower.contains('grading') ||
        lower.contains('size difference') ||
        lower.contains('shooters')) {
      return AquaDocMessage(
        id: 'ai-${now.millisecondsSinceEpoch}',
        text: '🐟 Husbandry Clinical Advisory: Cannibalism & Size Grading Protocol\n\n'
            'Target Subject: $species Fingerlings & Juveniles in $unitName\n'
            'Severity: PRODUCTION EFFICIENCY & INJURY RISK\n\n'
            '📋 Root Cause Analysis:\n'
            '• African Catfish (Clarias gariepinus) exhibit aggressive predatory hierarchy ("shooters") when size variation exceeds 1.5x.\n'
            '• Triggered by underfeeding, wide pellet size variance, or long sorting intervals.\n\n'
            '💉 Corrective Husbandry Protocol:\n'
            '1. Immediate Size Grading: Mechanically sort the cohort into 3 size classes (Large, Medium, Small) using floating grading boxes.\n'
            '2. Feed Frequency: Increase feeding frequency to 3–4 times daily for fingerlings (9:00 AM, 1:00 PM, 5:00 PM, 9:00 PM).\n'
            '3. Feed Satiation: Ensure feeding reaches satiation (until 95% of active feeding frenzy subsides).\n'
            '4. Salt Dressing: Add 2.0 ppt salt to tanks post-grading to heal bite abrasions and prevent secondary Aeromonas / Saprolegnia invasion.',
        isUser: false,
        timestamp: now,
        severity: 'info',
      );
    }

    // Default: Comprehensive Clinical Assessment Tailored to Live Farm Metrics
    return AquaDocMessage(
      id: 'ai-${now.millisecondsSinceEpoch}',
      text: '🏥 AquaDoc Clinical Diagnostic Summary for $unitName\n\n'
          'Species: $species\n'
          'Live Telemetry: Water Temperature $tempStr | Monitoring Status: Active\n\n'
          '📋 Clinical Evaluation:\n'
          'Based on current operational telemetry, your aquaculture environment is within normal physiological limits for $species.\n\n'
          '💡 Clinical Guidance for Maximum Growth & Disease Prevention:\n'
          '1. Feeding Precision: Distribute feed evenly across the pond surface to avoid density crowding and physical abrasions.\n'
          '2. Water Monitoring: Keep Dissolved Oxygen > 4.5 mg/L, Total Ammonia Nitrogen < 1.0 mg/L, and pH between 6.8 – 8.0.\n'
          '3. Health Inspection: Conduct daily visual audits during the morning feed response for signs of skin discoloration, fin erosion, or erratic swimming.\n\n'
          '💬 You can ask me specific questions regarding any symptoms (e.g. "red sores", "white spots", "swollen belly", "fin rot", "piping at surface") or chemical dosage calculations.',
      isUser: false,
      timestamp: now,
      severity: 'info',
      missingData: const ['Nitrite NO2', 'Alkalinity', 'Water Hardness'],
    );
  }

  void _scrollToBottom() {
    Future.delayed(const Duration(milliseconds: 100), () {
      if (_scrollController.hasClients) {
        _scrollController.animateTo(
          _scrollController.position.maxScrollExtent,
          duration: const Duration(milliseconds: 300),
          curve: Curves.easeOut,
        );
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Row(
          children: [
            Icon(Icons.psychology, color: Colors.blueAccent),
            SizedBox(width: 8),
            Text('AquaDoc AI Advisor'),
          ],
        ),
        actions: [
          IconButton(
            icon: const Icon(Icons.medical_services_outlined, color: Colors.redAccent),
            tooltip: 'Open Disease Triage',
            onPressed: () => context.go('/triage'),
          ),
          Container(
            margin: const EdgeInsets.only(right: 12, top: 10, bottom: 10),
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
            decoration: BoxDecoration(
              color: Colors.green.withOpacity(0.15),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: Colors.green.withOpacity(0.3)),
            ),
            child: const Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(Icons.circle, size: 8, color: Colors.green),
                SizedBox(width: 4),
                Text('RAG ACTIVE', style: TextStyle(color: Colors.green, fontSize: 10, fontWeight: FontWeight.bold)),
              ],
            ),
          ),
        ],
      ),
      body: Column(
        children: [
          // Live Injected Context Strip
          Builder(builder: (context) {
            final farmUnits = ref.watch(farmUnitsProvider).units;
            final sensorData = ref.watch(sensorDataProvider).currentData;
            final activeUnit = farmUnits.firstOrNull;

            final pondLabel = activeUnit != null ? activeUnit.name : 'No Ponds Set';
            final pondValue = activeUnit != null ? '${activeUnit.species.split(" ").first} (${activeUnit.totalBiomassKg.toStringAsFixed(0)}kg)' : 'Manual Setup';
            
            final doValue = activeUnit?.manualDO != null
                ? '${activeUnit!.manualDO!.toStringAsFixed(1)} mg/L (Lab)'
                : '-- mg/L (Unmeasured)';
            
            final tempValue = (sensorData != null && sensorData.waterTemperature > 0)
                ? '${sensorData.waterTemperature.toStringAsFixed(1)}°C (Live)'
                : (activeUnit?.manualTemp != null
                    ? '${activeUnit!.manualTemp!.toStringAsFixed(1)}°C (Lab)'
                    : '-- °C');
            
            final tanValue = activeUnit?.manualTAN != null
                ? '${activeUnit!.manualTAN!.toStringAsFixed(2)} mg/L (Lab)'
                : '-- mg/L (Unmeasured)';

            return Container(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
              color: Colors.grey.withOpacity(0.08),
              child: SingleChildScrollView(
                scrollDirection: Axis.horizontal,
                child: Row(
                  children: [
                    _ContextChip(icon: Icons.waves, label: pondLabel, value: pondValue),
                    const SizedBox(width: 8),
                    _ContextChip(icon: Icons.air, label: 'DO', value: doValue),
                    const SizedBox(width: 8),
                    _ContextChip(icon: Icons.thermostat, label: 'Temp', value: tempValue),
                    const SizedBox(width: 8),
                    _ContextChip(icon: Icons.science, label: 'TAN', value: tanValue),
                    const SizedBox(width: 8),
                    _ContextChip(
                      icon: Icons.warning_amber,
                      label: 'Missing Tests',
                      value: activeUnit?.manualDO == null ? 'DO, TAN, Alk' : 'Alkalinity, NO2',
                      isWarning: true,
                    ),
                  ],
                ),
              ),
            );
          }),

          // Message List
          Expanded(
            child: ListView.builder(
              controller: _scrollController,
              padding: const EdgeInsets.all(16),
              itemCount: _messages.length,
              itemBuilder: (context, index) {
                final msg = _messages[index];
                return _buildMessageBubble(context, msg);
              },
            ),
          ),

          // Typing indicator
          if (_isTyping)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 6),
              child: Row(
                children: [
                  const SizedBox(
                    width: 14,
                    height: 14,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  ),
                  const SizedBox(width: 10),
                  Text(
                    'AquaDoc is analyzing your request...',
                    style: TextStyle(fontSize: 12, color: Colors.grey[600], fontStyle: FontStyle.italic),
                  ),
                ],
              ),
            ),

          // Quick prompt chips
          Container(
            height: 38,
            margin: const EdgeInsets.symmetric(vertical: 4),
            child: ListView.separated(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              scrollDirection: Axis.horizontal,
              itemCount: _quickPrompts.length,
              separatorBuilder: (_, __) => const SizedBox(width: 8),
              itemBuilder: (context, index) {
                final prompt = _quickPrompts[index];
                return ActionChip(
                  label: Text(prompt, style: const TextStyle(fontSize: 12)),
                  onPressed: () => _sendMessage(prompt),
                );
              },
            ),
          ),

          // Input Bar
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: Theme.of(context).scaffoldBackgroundColor,
              border: Border(top: BorderSide(color: Colors.grey.withOpacity(0.2))),
            ),
            child: SafeArea(
              child: Row(
                children: [
                  Expanded(
                    child: TextField(
                      controller: _textController,
                      decoration: const InputDecoration(
                        hintText: 'Ask AquaDoc about pond health, DO, feeding...',
                        border: OutlineInputBorder(
                          borderRadius: BorderRadius.all(Radius.circular(24)),
                        ),
                        contentPadding: EdgeInsets.symmetric(horizontal: 16, vertical: 10),
                      ),
                      onSubmitted: _sendMessage,
                    ),
                  ),
                  const SizedBox(width: 8),
                  IconButton.filled(
                    onPressed: () => _sendMessage(_textController.text),
                    icon: const Icon(Icons.send_rounded),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildMessageBubble(BuildContext context, AquaDocMessage msg) {
    if (msg.isUser) {
      return Align(
        alignment: Alignment.centerRight,
        child: Container(
          margin: const EdgeInsets.only(bottom: 12, left: 48),
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          decoration: BoxDecoration(
            color: Theme.of(context).colorScheme.primary,
            borderRadius: const BorderRadius.only(
              topLeft: Radius.circular(16),
              topRight: Radius.circular(16),
              bottomLeft: Radius.circular(16),
            ),
          ),
          child: Text(
            msg.text,
            style: const TextStyle(color: Colors.white, fontSize: 14),
          ),
        ),
      );
    }

    return Align(
      alignment: Alignment.centerLeft,
      child: Container(
        margin: const EdgeInsets.only(bottom: 16, right: 24),
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: Theme.of(context).cardColor,
          borderRadius: const BorderRadius.only(
            topLeft: Radius.circular(16),
            topRight: Radius.circular(16),
            bottomRight: Radius.circular(16),
          ),
          border: Border.all(color: Colors.grey.withOpacity(0.15)),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withOpacity(0.04),
              blurRadius: 8,
              offset: const Offset(0, 2),
            ),
          ],
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Icon(Icons.psychology, color: Colors.blueAccent, size: 18),
                const SizedBox(width: 6),
                const Text('AquaDoc Consultant', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 13)),
                const Spacer(),
                Text(
                  '${msg.timestamp.hour.toString().padLeft(2, '0')}:${msg.timestamp.minute.toString().padLeft(2, '0')}',
                  style: TextStyle(fontSize: 10, color: Colors.grey[500]),
                ),
              ],
            ),
            const SizedBox(height: 10),
            Text(
              msg.text,
              style: const TextStyle(fontSize: 14, height: 1.4),
            ),

            // Missing Data Warnings
            if (msg.missingData.isNotEmpty) ...[
              const SizedBox(height: 12),
              Wrap(
                spacing: 6,
                runSpacing: 4,
                children: msg.missingData.map((param) {
                  return Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                    decoration: BoxDecoration(
                      color: Colors.amber.withOpacity(0.15),
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(color: Colors.amber.withOpacity(0.4)),
                    ),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        const Icon(Icons.help_outline, size: 12, color: Colors.amber),
                        const SizedBox(width: 4),
                        Text(
                          '$param: UNKNOWN',
                          style: TextStyle(fontSize: 10, color: Colors.amber[900], fontWeight: FontWeight.bold),
                        ),
                      ],
                    ),
                  );
                }).toList(),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class _ContextChip extends StatelessWidget {
  final IconData icon;
  final String label;
  final String value;
  final bool isWarning;

  const _ContextChip({
    required this.icon,
    required this.label,
    required this.value,
    this.isWarning = false,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: isWarning ? Colors.amber.withOpacity(0.2) : Colors.blue.withOpacity(0.1),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 12, color: isWarning ? Colors.amber[900] : Colors.blue[700]),
          const SizedBox(width: 4),
          Text(
            '$label: $value',
            style: TextStyle(
              fontSize: 11,
              fontWeight: FontWeight.w600,
              color: isWarning ? Colors.amber[900] : Colors.blue[800],
            ),
          ),
        ],
      ),
    );
  }
}
