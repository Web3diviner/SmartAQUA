# Smart Aqua Data Model

## Design Goal

Move from a device-centric product to a farm-centric aquaculture platform while retaining existing device and feeding models.

## Core Domain Hierarchy

```text
User
  |
  +-- Farm
       |
       +-- Pond
            |
            +-- ProductionCycle
            |     |
            |     +-- FishStock
            |     +-- SamplingRecord
            |     +-- MortalityRecord
            |     +-- DiseaseCase
            |
            +-- Device
                  |
                  +-- Sensor
                  +-- FeedingEvent
```

## Recommended Core Entities

### farms

```text
id
owner_user_id
name
location
timezone
status
created_at
updated_at
```

### farm_members

```text
id
farm_id
user_id
role
status
created_at
```

Roles:

- owner
- manager
- worker
- consultant
- veterinarian
- viewer

### ponds

```text
id
farm_id
name
type
volume_liters
location_description
status
created_at
updated_at
```

`type` examples:

- concrete_tank
- earthen_pond
- tarpaulin
- cage
- raceway

### production_cycles

```text
id
pond_id
species_id
stocking_date
initial_population
current_population
initial_average_weight_g
current_average_weight_g
status
expected_harvest_date
actual_harvest_date
created_at
updated_at
```

### fish_stock

Use if multiple stock groups/species may share a cycle.

```text
id
production_cycle_id
species_id
strain
count
average_weight_g
source_hatchery
stocked_at
status
```

### sampling_records

```text
id
production_cycle_id
sample_date
sample_size
average_weight_g
average_length_cm
estimated_biomass_kg
condition_factor
notes
recorded_by
```

### mortality_records

```text
id
production_cycle_id
recorded_at
count
suspected_cause
confirmed_cause
notes
recorded_by
```

## Device Domain

Existing device model remains.

Add:

```text
pond_id nullable
device_type
hardware_revision
provisioning_status
security_status
```

## Sensor Domain

### sensors

```text
id
device_id
pond_id
sensor_type
model
unit
serial_number
installed_at
status
last_calibrated_at
next_calibration_due
```

### sensor_readings

```text
id
sensor_id
pond_id
recorded_at
value
quality_flag
confidence
source
raw_value
metadata_json
```

`quality_flag`:

- valid
- estimated
- suspect
- invalid
- stale

`source`:

- device
- manual
- import
- derived

### sensor_calibrations

```text
id
sensor_id
calibrated_at
calibration_type
reference_values_json
slope
offset
temperature_compensation
performed_by
notes
```

## Feeding Domain

Keep existing feeding tables.

Recommended future additions:

### feed_inventory

```text
id
farm_id
feed_brand
feed_type
pellet_size_mm
quantity_kg
unit_cost
batch_number
expiry_date
```

### ration_recommendations

Prefer using the unified recommendation model below.

## Recommendation Domain

### recommendations

```text
id
farm_id
pond_id
production_cycle_id
type
source
title
reason
confidence
severity
status
requires_approval
proposed_action_json
evidence_json
model_provenance_json
expires_at
created_at
approved_at
approved_by
rejected_at
rejected_by
```

Status:

- pending
- approved
- rejected
- expired
- executed
- failed

## Command Domain

### device_commands

```text
id
device_id
recommendation_id nullable
command_type
payload_json
requested_by
approved_by
status
created_at
sent_at
acknowledged_at
completed_at
error_message
```

## Disease Domain

### disease_cases

```text
id
production_cycle_id
opened_at
status
severity
reported_symptoms_json
mortality_count
aquadoc_assessment_json
aquadoc_confidence
expert_required
expert_diagnosis
resolved_at
outcome
```

### disease_case_images

```text
id
disease_case_id
storage_url
captured_at
uploaded_by
image_type
```

### disease_assessments

```text
id
disease_case_id
source
assessment_json
confidence
evidence_json
created_at
```

Sources:

- aquadoc
- expert
- laboratory
- farmer

## Expert Domain

### experts

```text
id
user_id
specialty
verification_status
credentials_json
rating
active
```

### consultations

```text
id
farmer_user_id
expert_id
farm_id
pond_id
disease_case_id nullable
status
payment_status
started_at
completed_at
summary
```

### consultation_messages

```text
id
consultation_id
sender_user_id
message
attachment_url
created_at
```

## AquaDoc Domain

### aquadoc_conversations

```text
id
user_id
farm_id nullable
pond_id nullable
production_cycle_id nullable
created_at
updated_at
```

### aquadoc_messages

```text
id
conversation_id
role
content
structured_payload_json
created_at
```

### aquadoc_memories

Store only useful structured context.

```text
id
conversation_id
memory_type
memory_json
valid_from
valid_until
created_at
```

Avoid treating unlimited chat transcripts as authoritative memory.

## Knowledge Domain

### knowledge_documents

```text
id
title
source
author
year
document_type
species
topic
region
evidence_level
review_status
file_url
checksum
created_at
```

### knowledge_chunks

```text
id
document_id
chunk_index
content
page_number
section
metadata_json
embedding
created_at
```

## Audit Domain

### audit_logs

```text
id
actor_type
actor_id
action
resource_type
resource_id
before_json
after_json
ip_address
user_agent
created_at
```

All high-risk operations must be auditable.
