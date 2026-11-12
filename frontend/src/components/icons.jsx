// The handful of icons the app uses, drawn in one style: 1.6 stroke, round ends.

function Icon({ children, className = "h-5 w-5", ...rest }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.6}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      className={className}
      {...rest}
    >
      {children}
    </svg>
  );
}

export const SearchIcon = (props) => (
  <Icon {...props}>
    <circle cx="11" cy="11" r="6.5" />
    <path d="m20 20-4.2-4.2" />
  </Icon>
);

export const UploadIcon = (props) => (
  <Icon {...props}>
    <path d="M12 16V4m0 0L7.5 8.5M12 4l4.5 4.5" />
    <path d="M4 15.5v2A2.5 2.5 0 0 0 6.5 20h11a2.5 2.5 0 0 0 2.5-2.5v-2" />
  </Icon>
);

export const DownloadIcon = (props) => (
  <Icon {...props}>
    <path d="M12 4v12m0 0 4.5-4.5M12 16l-4.5-4.5" />
    <path d="M4 15.5v2A2.5 2.5 0 0 0 6.5 20h11a2.5 2.5 0 0 0 2.5-2.5v-2" />
  </Icon>
);

export const CloseIcon = (props) => (
  <Icon {...props}>
    <path d="m6 6 12 12M18 6 6 18" />
  </Icon>
);

export const CheckIcon = (props) => (
  <Icon {...props}>
    <path d="m5 12.5 4.5 4.5L19 7.5" />
  </Icon>
);

export const AlertIcon = (props) => (
  <Icon {...props}>
    <path d="M12 4.5 3.5 19h17L12 4.5Z" />
    <path d="M12 10v4.5M12 17v.01" />
  </Icon>
);

export const ClockIcon = (props) => (
  <Icon {...props}>
    <circle cx="12" cy="12" r="8" />
    <path d="M12 8v4.5l3 1.5" />
  </Icon>
);

export const RetryIcon = (props) => (
  <Icon {...props}>
    <path d="M19 12a7 7 0 1 1-2.3-5.2" />
    <path d="M19 4.5V8h-3.5" />
  </Icon>
);

export const CodeIcon = (props) => (
  <Icon {...props}>
    <path d="m8.5 8-4 4 4 4M15.5 8l4 4-4 4M13.5 5.5l-3 13" />
  </Icon>
);

export const PhotoIcon = (props) => (
  <Icon {...props}>
    <rect x="3.5" y="5" width="17" height="14" rx="2.5" />
    <circle cx="9" cy="10" r="1.6" />
    <path d="m4 17 4.8-4.3a1.5 1.5 0 0 1 2 0L14 15.5l1.7-1.5a1.5 1.5 0 0 1 2 0L20 16" />
  </Icon>
);

export const PinIcon = (props) => (
  <Icon {...props}>
    <path d="M9 4h6l-1 6 3 3H7l3-3-1-6ZM12 13v7" />
  </Icon>
);

/** Something is happening and nobody knows for how long. */
export function Spinner({ className = "h-4 w-4" }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true" className={`animate-spin ${className}`}>
      <circle cx="12" cy="12" r="9" stroke="currentColor" strokeOpacity=".2" strokeWidth="2.5" />
      <path d="M21 12a9 9 0 0 0-9-9" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" />
    </svg>
  );
}

/** The PrimeDrive mark: a photo corner with a sun. */
export function Logo({ className = "h-7 w-7" }) {
  return (
    <svg viewBox="0 0 32 32" aria-hidden="true" className={className}>
      <rect width="32" height="32" rx="9" fill="#5DC9E2" />
      <circle cx="21.5" cy="10.5" r="3" fill="#04252E" />
      <path d="M5 24.5 12.2 15a1.6 1.6 0 0 1 2.5 0l4 5.2 1.9-2.3a1.6 1.6 0 0 1 2.5 0L27 22.5V23a4 4 0 0 1-4 4H9a4 4 0 0 1-4-2.5Z" fill="#04252E" />
    </svg>
  );
}
