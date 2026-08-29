# Recirculating Aquaculture Systems (RAS) Design, Biofilter Nitrification, and Operational Management in Tropical West Africa

**Document ID:** NG-AQUA-MANUAL-006  
**Publisher / Source:** WorldFish, West Africa Agricultural Productivity Programme (WAAPP) & African Aquaculture Engineering Institute  
**Year:** 2024  
**Topics:** ras, recirculating_aquaculture, biofilter, mbbr, nitrification, alkalinity, catfish, water_quality, energy_backup  
**Evidence Level:** A_OFFICIAL_GUIDELINE  

---

## Section 1: RAS Engineering Principles and Component Sizing

Recirculating Aquaculture Systems (RAS) allow high-density fish production (80 – 120 kg/$m^3$ for *Clarias gariepinus*) in peri-urban areas of Lagos, Ibadan, Port Harcourt, and Abuja where land and water availability are constrained.

### Core Loop Process Sequence:
$$\text{Culture Tanks} \longrightarrow \text{Solids Separation (Settler/Drum)} \longrightarrow \text{Biofilter (MBBR/Trickling)} \longrightarrow \text{Degassing / $CO_2$ Stripping} \longrightarrow \text{Aeration/Oxygenation} \longrightarrow \text{UV Sterilization} \longrightarrow \text{Return Pump}$$

### 1. Mechanical Solids Removal
- **Radial Flow Settlers & Swirl Separators:** Capture 70% – 85% of settleable fecal solids and uneaten pellets. Siphon bottom sludge every 6–12 hours.
- **Rotary Drum Screen Filters:** 60 μm – 80 μm screen mesh size is recommended for commercial grow-out systems. Removing feces *before* mineralization reduces biological oxygen demand (BOD) and prevents ammonia spikes in the biofilter.

---

## Section 2: Biological Filter Nitrification Stoichiometry & Sizing

The biofilter converts toxic un-ionized ammonia ($NH_3$) into non-toxic nitrate ($NO_3^-$) through a two-step aerobic bacterial oxidation:

1. **Ammonia Oxidation:**
   $$NH_4^+ + 1.5\,O_2 \xrightarrow{\text{Nitrosomonas}} NO_2^- + H_2O + 2\,H^+ + \text{Energy}$$
2. **Nitrite Oxidation:**
   $$NO_2^- + 0.5\,O_2 \xrightarrow{\text{Nitrobacter / Nitrospira}} NO_3^- + \text{Energy}$$

### Critical Nitrification Parameters & Alkalinity Balancing:
- **Oxygen Demand:** Oxidation of 1.0 g Total Ammonia Nitrogen (TAN) requires **4.57 g of Dissolved Oxygen ($O_2$)**. Biofilter dissolved oxygen must remain strictly above **4.0 mg/L**.
- **Alkalinity Consumption:** Nitrification produces $H^+$ ions, consuming **7.14 g of $CaCO_3$ alkalinity per 1.0 g TAN oxidized**.
- **Daily Dosing Recommendation:** In closed RAS, dose **Sodium Bicarbonate ($NaHCO_3$)** at a rate of **150 – 250 g per 1.0 kg feed administered daily** to maintain total alkalinity at **100 – 150 mg/L as $CaCO_3$** and keep system pH buffered between **7.0 and 7.6**.

---

## Section 3: Moving Bed Biofilm Reactor (MBBR) Sizing Guidelines

For African Catfish fed a 42% Crude Protein diet (producing ~30–35 g TAN per 1.0 kg feed):

| Design Metric | Recommended Value | Engineering Note |
|---|---|---|
| **Specific Surface Area (SSA)** | 500 – 800 $m^2/m^3$ | Using Virgin HDPE K1 / K3 Kaldnes-type media. |
| **Media Filling Fraction** | 50% – 60% of chamber volume | Allows free fluidization by aeration grids. |
| **Volumetric TAN Removal Rate (VTR)** | 0.40 – 0.60 $g\text{ TAN} / m^2 \cdot \text{day}$ at 28°C | Tropical warm-water nitrification kinetics. |
| **Hydraulic Retention Time (HRT)** | 20 – 35 minutes | Ensures sufficient contact time across biofilm. |
| **Biofilter Air Supply** | 1.5 – 2.0 $m^3 \text{ air} / m^3 \text{ water / hour}$ | Provides aeration and media circular agitation. |

---

## Section 4: Biofilter Maturation & Seeding Protocol

New RAS biofilters require 21 to 30 days of biological seeding before stocking commercial fish biomass:

1. **Step 1 (Inoculation):** Fill system with dechlorinated water at 27°C – 29°C. Add commercial nitrifying bacterial concentrate or 5% mature biofilter media from an established disease-free farm.
2. **Step 2 (Ammonia Loading):** Dose pure ammonium chloride ($NH_4Cl$) or food-grade urea to achieve 3.0 – 5.0 mg/L TAN.
3. **Step 3 (Nitrification Curve Monitoring):**
   - **Days 1 – 8:** Ammonia rises and peaks; *Nitrosomonas* population establishes.
   - **Days 9 – 18:** Ammonia drops; Nitrite ($NO_2^-$) spikes to > 5.0 mg/L as *Nitrobacter* develops.
   - **Days 19 – 28:** Nitrite crashes to < 0.2 mg/L; Nitrate ($NO_3^-$) accumulates.
4. **Step 4 (Completion Validation):** When the biofilter can completely convert 3.0 mg/L added TAN to $NO_3^-$ in under 24 hours with zero nitrite, the system is fully matured and ready for full stocking.

---

## Section 5: Emergency Power Interlocking and Failure Recovery

In Nigerian aquaculture, electrical grid outages represent the single highest mortality risk in intensive RAS.

### Emergency Protocols:
1. **Automatic Transfer Switch (ATS):** Interlocks main grid with standby diesel generator within **30–60 seconds**.
2. **Dedicated DC Battery Inverter Aerators:** Submerged 12V/24V blowers or air stones running in culture tanks immediately supply emergency aeration if the generator fails to crank.
3. **Emergency Feeding Halt:** If water circulation or aeration stops, immediately stop all feeding. Fish under digestion consume 300% more oxygen and quickly succumb to asphyxiation.
