# API Contract (Current)

Single `api` package with handlers composed in `router.go`. All routes are under `/api/v1` unless noted.

## Auth
- `POST /auth/login` — body `{"email","password"}`; returns `{token}` (alias of `access_token`).
- `POST /auth/signup` — body `{"email","password","fullName"}`; returns `{token}`.
- `POST /auth/logout` — Bearer token; returns message.
- `POST /auth/forgot-password` — body `{"email"}`; returns generic success.
- `POST /auth/reset-password` — body `{"token","newPassword"}`.
- `PUT /auth/update-password` — Bearer token; body `{"currentPassword","newPassword"}`.
- `GET /auth/me` — Bearer token; returns user profile.

## Public
- `POST /submissions` (alias: `/submit`) — create submission; returns submission summary.

## User (auth required)
- `GET /submissions` — list current user's submissions.
- `GET /submissions/:id` — get submission detail (includes enrichment/analysis/report IDs/status when present).
- `GET /submissions/:id/enrichment` — enrichment detail (user-scoped).
- `GET /submissions/:id/analysis` — analysis detail (user-scoped).
- `GET /submissions/:id/report/preview` — preview report (auth).
- `GET /submissions/:id/report/download` — download report (auth).
- `PUT /user/profile` — update profile.
- `DELETE /user` — deactivate account (soft delete).

## Admin (auth + admin role)
- `GET /admin/submissions` — list all submissions (filters: status, email, pagination).
- `GET /admin/submissions/:id` — submission detail.
- `GET /admin/submissions/:id/enrichment` — enrichment by submission ID.
- `POST /admin/submissions/:id/retry-enrichment` — enqueue enrichment retry.
- `POST /admin/submissions/:id/retry-analysis` — enqueue analysis retry.
- `GET /admin/enrichment/:id` — enrichment detail by enrichment ID.
- `PUT /admin/enrichment/:id` — update enrichment.
- `POST /admin/enrichment/:id/approve` — approve enrichment (creates analysis).
- `POST /admin/enrichment/:id/unlock` — unlock enrichment for edits.
- `GET /admin/analysis/:id` — analysis detail.
- `PUT /admin/analysis/:id` — update analysis.
- `POST /admin/analysis/:id/version` — create new analysis version.
- `POST /admin/analysis/:id/approve` — approve analysis (triggers report).
- `POST /admin/analysis/:id/send` — mark analysis sent to user.
- `GET /admin/analytics` — analytics summary.

## Middleware expectations
- Auth: JWT HMAC (`SUPABASE_JWT_SECRET`); role fetched from `user_profiles`.
- Admin routes require `userRole` in {admin, super_admin, service_role}.
- Rate limiting: global request limiter + auth-specific limiter.
- CORS: configured via `allowedOrigins`.

## Response shapes (key types)
- `ErrorResponse {error,message}`.
- `SubmissionResponse {id,status,...}` (submission status is always `received`; workflow status comes from enrichment/analysis/report).
- `EnrichmentResponse` — includes status/current_step/content.
- `AnalysisResponse` — includes status/version/content.
- `ReportResponse` — includes status/contentURL.
- `UserProfileResponse {user: {id,email,fullName,role,isActive,...}}`.

## Auth mock mode (for local testing)
- Set `MOCK_AUTH=true` or `SUPABASE_URL=mock` to bypass Supabase, auto-provision users in `user_profiles`, and mint local JWTs signed with `SUPABASE_JWT_SECRET`.
