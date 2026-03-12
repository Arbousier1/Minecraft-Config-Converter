import { createTheme } from "@mui/material/styles";

export const appTheme = createTheme({
  palette: {
    mode: "light",
    primary: {
      main: "#0b57d0"
    },
    secondary: {
      main: "#44546a"
    },
    success: {
      main: "#107c10"
    },
    error: {
      main: "#c42b1c"
    },
    background: {
      default: "#dfe3e8",
      paper: "#ffffff"
    },
    text: {
      primary: "#1f2328",
      secondary: "#66717f"
    },
    divider: "rgba(31, 35, 40, 0.12)",
    action: {
      selected: "rgba(11, 87, 208, 0.08)"
    },
    window: "#f5f6f7",
    sidebar: "#f0f2f4",
    panel: {
      header: "#f6f7f9"
    }
  },
  shape: {
    borderRadius: 6
  },
  typography: {
    fontFamily: '"Segoe UI Variable", "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif',
    h3: {
      fontSize: "1.1rem",
      fontWeight: 600,
      lineHeight: 1.3
    },
    h4: {
      fontSize: "1rem",
      fontWeight: 600,
      lineHeight: 1.35
    },
    h5: {
      fontSize: "0.95rem",
      fontWeight: 600
    },
    subtitle2: {
      fontSize: "0.72rem",
      letterSpacing: "0.04em",
      textTransform: "none"
    },
    button: {
      fontSize: "0.82rem"
    }
  },
  components: {
    MuiCard: {
      styleOverrides: {
        root: {
          border: "1px solid rgba(31, 35, 40, 0.12)",
          boxShadow: "none",
          borderRadius: 6
        }
      }
    },
    MuiButton: {
      defaultProps: {
        disableElevation: true
      },
      styleOverrides: {
        root: {
          borderRadius: 4,
          fontWeight: 600,
          paddingInline: 14,
          minHeight: 32,
          textTransform: "none"
        }
      }
    },
    MuiChip: {
      styleOverrides: {
        root: {
          borderRadius: 4,
          height: 24
        }
      }
    },
    MuiOutlinedInput: {
      styleOverrides: {
        root: {
          borderRadius: 4,
          backgroundColor: "#fff"
        }
      }
    },
    MuiAlert: {
      styleOverrides: {
        root: {
          borderRadius: 4
        }
      }
    }
  }
});
