# Data migrations

The current mobile client stores its profile, credentials, playback history, and local preferences on-device. The Go music-source service is therefore stateless for now.

When server-side accounts and cross-device library sync are enabled, versioned SQLite migrations will be added here for users, playlists, favourites, and listening history. Do not add unversioned schema changes to application startup.
