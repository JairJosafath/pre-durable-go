// Local development only. When NEXT_PUBLIC_LOCAL_USER names a user, the app
// starts signed in as that user and never shows the sign-in form. The local
// API (`make local`) has no Cognito: it takes the token to be the user's
// name. Against AWS this variable must stay unset; a deployed API would
// reject such a token, so setting it by mistake cannot let anyone in.
const localUser = process.env.NEXT_PUBLIC_LOCAL_USER;

/** The session of the local user, or null when users sign in through Cognito. */
export const localAuth = localUser ? { username: localUser, authenticated: true, idToken: localUser } : null;
