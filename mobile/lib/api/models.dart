// Data models mirroring the Go API's JSON.

double _d(dynamic v) => (v as num?)?.toDouble() ?? 0;
double? _dn(dynamic v) => (v as num?)?.toDouble();
int _i(dynamic v) => (v as num?)?.toInt() ?? 0;
int? _in(dynamic v) => (v as num?)?.toInt();
DateTime? _dt(dynamic v) => v == null ? null : DateTime.parse(v as String).toLocal();
List<T> _list<T>(dynamic v, T Function(Map<String, dynamic>) f) =>
    (v as List? ?? []).map((e) => f(e as Map<String, dynamic>)).toList();

class Skill {
  final int id;
  final String name;
  final int categoryId;
  final String categoryName;
  final bool verified;

  Skill.fromJson(Map<String, dynamic> j)
    : id = _i(j['id']),
      name = j['name'] as String,
      categoryId = _i(j['category_id']),
      categoryName = j['category_name'] as String? ?? '',
      verified = j['verified'] as bool? ?? false;
}

class Category {
  final int id;
  final String name;
  final String icon;
  final List<Skill> skills;

  Category.fromJson(Map<String, dynamic> j)
    : id = _i(j['id']),
      name = j['name'] as String,
      icon = j['icon'] as String? ?? '',
      skills = _list(j['skills'], Skill.fromJson);
}

class WorkerProfile {
  final String headline;
  final bool available;
  final int hourlyRate;
  final double radiusKm;
  final double rating;
  final int ratingCount;
  final int completedJobs;
  final double completionRate;
  final List<Skill> skills;
  final List<String> badges;

  WorkerProfile.fromJson(Map<String, dynamic> j)
    : headline = j['headline'] as String? ?? '',
      available = j['available'] as bool? ?? false,
      hourlyRate = _i(j['hourly_rate']),
      radiusKm = _d(j['radius_km']),
      rating = _d(j['rating']),
      ratingCount = _i(j['rating_count']),
      completedJobs = _i(j['completed_jobs']),
      completionRate = _d(j['completion_rate']),
      skills = _list(j['skills'], Skill.fromJson),
      badges = (j['badges'] as List? ?? []).cast<String>();
}

class User {
  final String id;
  final String firstName;
  final String lastName;
  final String username;
  final String avatarUrl;
  final String bio;
  final String zone;
  final bool isClient;
  final bool isWorker;
  final bool isAdmin;
  final bool identityVerified;
  final double clientRating;
  final int clientRatingCount;
  final WorkerProfile? worker;
  final String? email;
  final String? phone;
  final double? latitude;
  final double? longitude;

  User.fromJson(Map<String, dynamic> j)
    : id = j['id'] as String,
      firstName = j['first_name'] as String,
      lastName = j['last_name'] as String,
      username = j['username'] as String,
      avatarUrl = j['avatar_url'] as String? ?? '',
      bio = j['bio'] as String? ?? '',
      zone = j['zone'] as String? ?? '',
      isClient = j['is_client'] as bool? ?? false,
      isWorker = j['is_worker'] as bool? ?? false,
      isAdmin = j['is_admin'] as bool? ?? false,
      identityVerified = j['identity_verified'] as bool? ?? false,
      clientRating = _d(j['client_rating']),
      clientRatingCount = _i(j['client_rating_count']),
      worker = j['worker'] == null ? null : WorkerProfile.fromJson(j['worker'] as Map<String, dynamic>),
      email = j['email'] as String?,
      phone = j['phone'] as String?,
      latitude = _dn(j['latitude']),
      longitude = _dn(j['longitude']);

  String get fullName => '$firstName $lastName';
}

class PersonRef {
  final String id;
  final String firstName;
  final String lastName;
  final String avatarUrl;
  final double rating;
  final int ratingCount;
  final int completedJobs;

  PersonRef.fromJson(Map<String, dynamic> j)
    : id = j['id'] as String,
      firstName = j['first_name'] as String,
      lastName = j['last_name'] as String,
      avatarUrl = j['avatar_url'] as String? ?? '',
      rating = _d(j['rating']),
      ratingCount = _i(j['rating_count']),
      completedJobs = _i(j['completed_jobs']);

  String get fullName => '$firstName $lastName';
}

class Assignment {
  final DateTime? assignedAt;
  final double? workerDistanceKm;
  final DateTime? locationUpdatedAt;

  Assignment.fromJson(Map<String, dynamic> j)
    : assignedAt = _dt(j['assigned_at']),
      workerDistanceKm = _dn(j['worker_distance_km']),
      locationUpdatedAt = _dt(j['location_updated_at']);
}

class Payment {
  final int amount;
  final int platformFee;
  final int workerAmount;
  final int refundedAmount;
  final double commissionRate;
  final String status;

  Payment.fromJson(Map<String, dynamic> j)
    : amount = _i(j['amount']),
      platformFee = _i(j['platform_fee']),
      workerAmount = _i(j['worker_amount']),
      refundedAmount = _i(j['refunded_amount']),
      commissionRate = _d(j['commission_rate']),
      status = j['status'] as String;
}

class Application {
  final String id;
  final int price;
  final String message;
  final String status;
  final PersonRef? worker;
  final String headline;
  final double? distanceKm;

  Application.fromJson(Map<String, dynamic> j)
    : id = j['id'] as String,
      price = _i(j['price']),
      message = j['message'] as String? ?? '',
      status = j['status'] as String,
      worker = j['worker'] == null ? null : PersonRef.fromJson(j['worker'] as Map<String, dynamic>),
      headline = j['headline'] as String? ?? '',
      distanceKm = _dn(j['distance_km']);
}

class HistoryItem {
  final String status;
  final String note;
  final DateTime createdAt;

  HistoryItem.fromJson(Map<String, dynamic> j)
    : status = j['status'] as String,
      note = j['note'] as String? ?? '',
      createdAt = _dt(j['created_at'])!;
}

class Job {
  final String id;
  final String title;
  final String description;
  final int categoryId;
  final String categoryName;
  final String categoryIcon;
  final String? skillName;
  final String zone;
  final String pricingMode;
  final int budget;
  final int? agreedPrice;
  final DateTime? scheduledAt;
  final int durationMinutes;
  final List<String> photos;
  final String status;
  final DateTime createdAt;
  final PersonRef client;
  final PersonRef? worker;
  final int applicationsCount;
  final String myRole;
  final String? address;
  final double? latitude;
  final double? longitude;
  final double? distanceKm;
  final Assignment? assignment;
  final Payment? payment;
  final Application? myApplication;
  final bool reviewedByMe;
  final bool openReport;
  final bool hasConversation;
  final List<HistoryItem> history;

  Job.fromJson(Map<String, dynamic> j)
    : id = j['id'] as String,
      title = j['title'] as String,
      description = j['description'] as String? ?? '',
      categoryId = _i(j['category']['id']),
      categoryName = j['category']['name'] as String,
      categoryIcon = j['category']['icon'] as String? ?? '',
      skillName = j['skill_name'] as String?,
      zone = j['zone'] as String? ?? '',
      pricingMode = j['pricing_mode'] as String,
      budget = _i(j['budget']),
      agreedPrice = _in(j['agreed_price']),
      scheduledAt = _dt(j['scheduled_at']),
      durationMinutes = _i(j['duration_minutes']),
      photos = (j['photos'] as List? ?? []).cast<String>(),
      status = j['status'] as String,
      createdAt = _dt(j['created_at'])!,
      client = PersonRef.fromJson(j['client'] as Map<String, dynamic>),
      worker = j['worker'] == null ? null : PersonRef.fromJson(j['worker'] as Map<String, dynamic>),
      applicationsCount = _i(j['applications_count']),
      myRole = j['my_role'] as String? ?? 'candidate',
      address = j['address'] as String?,
      latitude = _dn(j['latitude']),
      longitude = _dn(j['longitude']),
      distanceKm = _dn(j['distance_km']),
      assignment = j['assignment'] == null ? null : Assignment.fromJson(j['assignment'] as Map<String, dynamic>),
      payment = j['payment'] == null ? null : Payment.fromJson(j['payment'] as Map<String, dynamic>),
      myApplication = j['my_application'] == null ? null : Application.fromJson(j['my_application'] as Map<String, dynamic>),
      reviewedByMe = j['reviewed_by_me'] as bool? ?? false,
      openReport = j['open_report'] as bool? ?? false,
      hasConversation = j['has_conversation'] as bool? ?? false,
      history = _list(j['history'], HistoryItem.fromJson);

  int get price => agreedPrice ?? budget;
  bool get isOpen => status == 'POSTED' || status == 'MATCHING';
}

class Message {
  final String id;
  final String jobId;
  final String? senderId;
  final String kind;
  final String body;
  final String attachmentUrl;
  final double? latitude;
  final double? longitude;
  final DateTime createdAt;

  Message.fromJson(Map<String, dynamic> j)
    : id = j['id'] as String,
      jobId = j['job_id'] as String? ?? '',
      senderId = j['sender_id'] as String?,
      kind = j['kind'] as String,
      body = j['body'] as String? ?? '',
      attachmentUrl = j['attachment_url'] as String? ?? '',
      latitude = _dn(j['latitude']),
      longitude = _dn(j['longitude']),
      createdAt = _dt(j['created_at'])!;
}

class Conversation {
  final String jobId;
  final String jobTitle;
  final String jobStatus;
  final PersonRef other;
  final String? lastMessage;
  final DateTime? lastAt;
  final bool closed;

  Conversation.fromJson(Map<String, dynamic> j)
    : jobId = j['job_id'] as String,
      jobTitle = j['job_title'] as String,
      jobStatus = j['job_status'] as String,
      other = PersonRef.fromJson(j['other'] as Map<String, dynamic>),
      lastMessage = j['last_message'] as String?,
      lastAt = _dt(j['last_at']),
      closed = j['closed'] as bool? ?? false;
}

class AppNotification {
  final String id;
  final String kind;
  final String title;
  final String body;
  final String? jobId;
  final bool read;
  final DateTime createdAt;

  AppNotification.fromJson(Map<String, dynamic> j)
    : id = j['id'] as String,
      kind = j['kind'] as String,
      title = j['title'] as String,
      body = j['body'] as String? ?? '',
      jobId = j['job_id'] as String?,
      read = j['read_at'] != null,
      createdAt = _dt(j['created_at'])!;
}

class WalletTx {
  final int amount;
  final String kind;
  final String description;
  final DateTime createdAt;

  WalletTx.fromJson(Map<String, dynamic> j)
    : amount = _i(j['amount']),
      kind = j['kind'] as String,
      description = j['description'] as String? ?? '',
      createdAt = _dt(j['created_at'])!;
}

class Wallet {
  final int balance;
  final int totalEarned;
  final int earnedThisMonth;
  final int pendingWithdrawals;
  final List<WalletTx> transactions;

  Wallet.fromJson(Map<String, dynamic> j)
    : balance = _i(j['balance']),
      totalEarned = _i(j['total_earned']),
      earnedThisMonth = _i(j['earned_this_month']),
      pendingWithdrawals = _i(j['pending_withdrawals']),
      transactions = _list(j['transactions'], WalletTx.fromJson);
}

class WorkerResult {
  final String id;
  final String firstName;
  final String lastName;
  final String avatarUrl;
  final String headline;
  final double rating;
  final int ratingCount;
  final int completedJobs;
  final int hourlyRate;
  final bool available;
  final double? distanceKm;
  final List<String> skills;

  WorkerResult.fromJson(Map<String, dynamic> j)
    : id = j['id'] as String,
      firstName = j['first_name'] as String,
      lastName = j['last_name'] as String,
      avatarUrl = j['avatar_url'] as String? ?? '',
      headline = j['headline'] as String? ?? '',
      rating = _d(j['rating']),
      ratingCount = _i(j['rating_count']),
      completedJobs = _i(j['completed_jobs']),
      hourlyRate = _i(j['hourly_rate']),
      available = j['available'] as bool? ?? false,
      distanceKm = _dn(j['distance_km']),
      skills = (j['skills'] as List? ?? []).cast<String>();
}

class Review {
  final String jobTitle;
  final String reviewerName;
  final int rating;
  final String comment;
  final DateTime createdAt;

  Review.fromJson(Map<String, dynamic> j)
    : jobTitle = j['job_title'] as String,
      reviewerName = j['reviewer_name'] as String,
      rating = _i(j['rating']),
      comment = j['comment'] as String? ?? '',
      createdAt = _dt(j['created_at'])!;
}
