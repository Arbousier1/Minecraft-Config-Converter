import { createTheme } from "@mui/material/styles";

export const appTheme = createTheme({
  palette: {
    mode: "light",
    primary: {
      main: "#0e7490"
    },
    secondary: {
      main: "#9a3412"
    },
    success: {
      main: "#15803d"
    },
    error: {
      main: "#c2410c"
    },
    background: {
      default: "#f6f2ea",
      paper: "#fffdf9"
    },
    text: {
      primary: "#0f172a",
      secondary: "#52606d"
    }
  },
  shape: {
    borderRadius: 20
  },
  typography: {
    fontFamily: '"IBM Plex Sans", "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif',
    h3: {
      fontSize: "clamp(1.9rem, 2.4vw, 2.8rem)",
      fontWeight: 700,
      lineHeight: 1.08
    },
    h4: {
      fontSize: "clamp(1.25rem, 1.5vw, 1.65rem)",
      fontWeight: 700,
      lineHeight: 1.15
    },
    h5: {
      fontSize: "1.05rem",
      fontWeight: 700
    },
    subtitle2: {
      fontSize: "0.72rem",
      letterSpacing: "0.12em",
      textTransform: "uppercase"
    }
  },
  components: {
    MuiCard: {
      styleOverrides: {
        root: {
          border: "1px solid rgba(15, 23, 42, 0.08)",
          boxShadow: "0 18px 38px rgba(15, 23, 42, 0.06)"
        }
      }
    },
    MuiButton: {
      defaultProps: {
        disableElevation: true
      },
      styleOverrides: {
        root: {
          borderRadius: 999,
          fontWeight: 700,
          paddingInline: 18
        }
      }
    },
    MuiChip: {
      styleOverrides: {
        root: {
          borderRadius: 999
        }
      }
    },
    MuiOutlinedInput: {
      styleOverrides: {
        root: {
          borderRadius: 16
        }
      }
    }
  }
});
