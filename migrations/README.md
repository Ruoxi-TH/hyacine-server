# Data migrations

The current mobile client stores its profile, credentials, playback history, and local preferences on-device. The Go music-source service is therefore stateless for now.

The App administration screen reads profile, credential-presence flags, favorites, listening history, and logs from the current device; it does not imply server-side persistence.

When server-side accounts and cross-device library sync are enabled, versioned SQLite migrations will be added here for users, playlists, favourites, listening history, and audit-safe administration metadata. JB persistence must not be added until its identity and credential model are defined. Do not add unversioned schema changes to application startup.
