// Plain-data models mirroring the backend JSON. Kept hand-rolled (no
// build_runner / freezed) to keep the build pipeline simple — the API
// surface is small enough.

class User {
  User({
    required this.id,
    required this.email,
    required this.fullName,
    required this.role,
    this.phone,
    this.birthDate,
    this.inviteCode,
    this.heightCm,
    this.chronicConditions,
    this.bpNorm,
    this.prescribedMeds,
    required this.onboarded,
    this.lang,
    required this.tz,
  });

  final String id;
  final String email;
  final String fullName;
  final String role; // patient | doctor | family
  final String? phone;
  final String? birthDate;
  final String? inviteCode;
  final int? heightCm;
  final String? chronicConditions;
  final String? bpNorm;
  final String? prescribedMeds;
  final bool onboarded;
  final String? lang;
  final String tz;

  factory User.fromJson(Map<String, dynamic> j) => User(
        id: j["id"] as String,
        email: j["email"] as String,
        fullName: j["full_name"] as String,
        role: j["role"] as String,
        phone: j["phone"] as String?,
        birthDate: j["birth_date"] as String?,
        inviteCode: j["invite_code"] as String?,
        heightCm: j["height_cm"] as int?,
        chronicConditions: j["chronic_conditions"] as String?,
        bpNorm: j["bp_norm"] as String?,
        prescribedMeds: j["prescribed_meds"] as String?,
        onboarded: j["onboarded"] as bool? ?? false,
        lang: j["lang"] as String?,
        tz: (j["tz"] as String?) ?? "UTC",
      );

  Map<String, dynamic> toJson() => {
        "id": id,
        "email": email,
        "full_name": fullName,
        "role": role,
        if (phone != null) "phone": phone,
        if (birthDate != null) "birth_date": birthDate,
        if (inviteCode != null) "invite_code": inviteCode,
        if (heightCm != null) "height_cm": heightCm,
        if (chronicConditions != null) "chronic_conditions": chronicConditions,
        if (bpNorm != null) "bp_norm": bpNorm,
        if (prescribedMeds != null) "prescribed_meds": prescribedMeds,
        "onboarded": onboarded,
        if (lang != null) "lang": lang,
        "tz": tz,
      };
}

class Metric {
  Metric({
    required this.id,
    required this.patientId,
    required this.kind,
    required this.value,
    this.value2,
    this.note,
    required this.measuredAt,
  });

  final String id;
  final String patientId;
  final String kind;
  final double value;
  final double? value2;
  final String? note;
  final DateTime measuredAt;

  factory Metric.fromJson(Map<String, dynamic> j) => Metric(
        id: j["id"] as String,
        patientId: j["patient_id"] as String,
        kind: j["kind"] as String,
        value: (j["value"] as num).toDouble(),
        value2: (j["value_2"] as num?)?.toDouble(),
        note: j["note"] as String?,
        measuredAt: DateTime.parse(j["measured_at"] as String).toLocal(),
      );
}

class Alert {
  Alert({
    required this.id,
    required this.patientId,
    this.metricId,
    required this.severity,
    required this.reason,
    required this.reasonCode,
    required this.algorithmVersion,
    required this.kind,
    this.value,
    this.baselineMean,
    this.baselineStd,
    required this.acknowledged,
    required this.createdAt,
  });

  final String id;
  final String patientId;
  final String? metricId;
  final String severity; // info | warning | critical
  final String reason;
  final String reasonCode;
  final String algorithmVersion;
  final String kind;
  final double? value;
  final double? baselineMean;
  final double? baselineStd;
  final bool acknowledged;
  final DateTime createdAt;

  factory Alert.fromJson(Map<String, dynamic> j) => Alert(
        id: j["id"] as String,
        patientId: j["patient_id"] as String,
        metricId: j["metric_id"] as String?,
        severity: j["severity"] as String,
        reason: (j["reason"] as String?) ?? "",
        reasonCode: (j["reason_code"] as String?) ?? "legacy",
        algorithmVersion: (j["algorithm_version"] as String?) ?? "v1",
        kind: j["kind"] as String,
        value: (j["value"] as num?)?.toDouble(),
        baselineMean: (j["baseline_mean"] as num?)?.toDouble(),
        baselineStd: (j["baseline_std"] as num?)?.toDouble(),
        acknowledged: j["acknowledged"] as bool? ?? false,
        createdAt: DateTime.parse(j["created_at"] as String).toLocal(),
      );
}

class Medication {
  Medication({
    required this.id,
    required this.patientId,
    required this.name,
    this.dosage,
    required this.timesOfDay,
    required this.startDate,
    this.endDate,
    required this.active,
    this.notes,
    required this.createdAt,
    this.prescribedBy,
    this.prescribedAt,
  });

  final String id;
  final String patientId;
  final String name;
  final String? dosage;
  final List<String> timesOfDay;
  final DateTime startDate;
  final DateTime? endDate;
  final bool active;
  final String? notes;
  final DateTime createdAt;
  final String? prescribedBy;
  final DateTime? prescribedAt;

  factory Medication.fromJson(Map<String, dynamic> j) => Medication(
        id: j["id"] as String,
        patientId: j["patient_id"] as String,
        name: j["name"] as String,
        dosage: j["dosage"] as String?,
        timesOfDay:
            ((j["times_of_day"] as List?) ?? const []).cast<String>(),
        startDate: DateTime.parse(j["start_date"] as String).toLocal(),
        endDate: (j["end_date"] as String?) != null
            ? DateTime.parse(j["end_date"] as String).toLocal()
            : null,
        active: j["active"] as bool? ?? true,
        notes: j["notes"] as String?,
        createdAt: DateTime.parse(j["created_at"] as String).toLocal(),
        prescribedBy: j["prescribed_by"] as String?,
        prescribedAt: (j["prescribed_at"] as String?) != null
            ? DateTime.parse(j["prescribed_at"] as String).toLocal()
            : null,
      );
}

class MedScheduleItem {
  MedScheduleItem({
    required this.medicationId,
    required this.name,
    this.dosage,
    required this.scheduledAt,
    required this.status,
  });

  final String medicationId;
  final String name;
  final String? dosage;
  final DateTime scheduledAt;
  final String status; // pending | taken | missed | skipped

  factory MedScheduleItem.fromJson(Map<String, dynamic> j) => MedScheduleItem(
        medicationId: j["medication_id"] as String,
        name: j["name"] as String,
        dosage: j["dosage"] as String?,
        scheduledAt: DateTime.parse(j["scheduled_at"] as String).toLocal(),
        status: j["status"] as String,
      );
}

class LinkedPatient {
  LinkedPatient({
    required this.patientId,
    required this.fullName,
    required this.email,
    this.phone,
    required this.relation,
  });

  final String patientId;
  final String fullName;
  final String email;
  final String? phone;
  final String relation;

  factory LinkedPatient.fromJson(Map<String, dynamic> j) => LinkedPatient(
        patientId: j["patient_id"] as String,
        fullName: j["full_name"] as String,
        email: j["email"] as String,
        phone: j["phone"] as String?,
        relation: j["relation"] as String,
      );
}

class Caregiver {
  Caregiver({
    required this.id,
    required this.fullName,
    required this.email,
    this.phone,
    required this.relation,
  });

  final String id;
  final String fullName;
  final String email;
  final String? phone;
  final String relation;

  factory Caregiver.fromJson(Map<String, dynamic> j) => Caregiver(
        id: j["id"] as String,
        fullName: j["full_name"] as String,
        email: j["email"] as String,
        phone: j["phone"] as String?,
        relation: j["relation"] as String,
      );
}

class Message {
  Message({
    required this.id,
    required this.senderId,
    required this.recipientId,
    required this.body,
    this.readAt,
    required this.createdAt,
    this.senderName,
  });

  final String id;
  final String senderId;
  final String recipientId;
  final String body;
  final DateTime? readAt;
  final DateTime createdAt;
  final String? senderName;

  factory Message.fromJson(Map<String, dynamic> j) => Message(
        id: j["id"] as String,
        senderId: j["sender_id"] as String,
        recipientId: j["recipient_id"] as String,
        body: j["body"] as String,
        readAt: (j["read_at"] as String?) != null
            ? DateTime.parse(j["read_at"] as String).toLocal()
            : null,
        createdAt: DateTime.parse(j["created_at"] as String).toLocal(),
        senderName: j["sender_name"] as String?,
      );
}

class Plan {
  Plan({
    required this.id,
    required this.patientId,
    required this.dayOfWeek,
    required this.title,
    required this.planType,
    this.timeOfDay,
    required this.createdAt,
  });

  final String id;
  final String patientId;
  final int dayOfWeek;
  final String title;
  final String planType;
  final String? timeOfDay;
  final DateTime createdAt;

  factory Plan.fromJson(Map<String, dynamic> j) => Plan(
        id: j["id"] as String,
        patientId: j["patient_id"] as String,
        dayOfWeek: j["day_of_week"] as int,
        title: j["title"] as String,
        planType: j["plan_type"] as String,
        timeOfDay: j["time_of_day"] as String?,
        createdAt: DateTime.parse(j["created_at"] as String).toLocal(),
      );
}

class CareNote {
  CareNote({
    required this.id,
    required this.patientId,
    required this.authorId,
    required this.authorName,
    required this.authorRole,
    required this.body,
    required this.createdAt,
  });

  final String id;
  final String patientId;
  final String authorId;
  final String authorName;
  final String authorRole;
  final String body;
  final DateTime createdAt;

  factory CareNote.fromJson(Map<String, dynamic> j) => CareNote(
        id: j["id"] as String,
        patientId: j["patient_id"] as String,
        authorId: j["author_id"] as String,
        authorName: j["author_name"] as String,
        authorRole: j["author_role"] as String,
        body: j["body"] as String,
        createdAt: DateTime.parse(j["created_at"] as String).toLocal(),
      );
}
