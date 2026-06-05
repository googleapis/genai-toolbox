document.addEventListener('DOMContentLoaded', function() {

  // Setup CSS for the wrapper and the banner
  var styleTag = document.createElement('style');
  styleTag.innerHTML = `
    .td-navbar .dropdown-menu {
      z-index: 9999 !important;
    }

    .theme-banner-wrapper {
      position: sticky;
      z-index: 20;
      padding-top: 15px;    /* This is your gap! */
      padding-bottom: 5px;  /* Breathing room below the banner */
      /* Uses Bootstrap's native body background variable, with white as fallback */
      background-color: var(--bs-body-bg, #ffffff);
    }

    .theme-migration-banner {
      background-color: #ebf3fc;
      border: 1px solid #80a7e9;
      color: #1c3a6b;
      border-radius: 4px;
      padding: 15px;
      text-align: center;
      width: 100%;
      box-shadow: 0 4px 6px rgba(0,0,0,0.05);
    }

    .theme-migration-banner a {
      color: #4484f4;
      text-decoration: underline;
      font-weight: bold;
    }

    /* DARK MODE STYLING */
    html[data-bs-theme="dark"] .theme-banner-wrapper,
    body.dark .theme-banner-wrapper,
    html.dark-mode .theme-banner-wrapper {
      /* Uses Docsy's dark mode background fallback if var fails */
      background-color: var(--bs-body-bg, #20252b);
    }

    html[data-bs-theme="dark"] .theme-migration-banner,
    body.dark .theme-migration-banner,
    html.dark-mode .theme-migration-banner {
      background-color: #1a273b;
      color: #e6efff;
      box-shadow: 0 4px 6px rgba(0,0,0,0.3);
    }

    html[data-bs-theme="dark"] .theme-migration-banner a,
    body.dark .theme-migration-banner a,
    html.dark-mode .theme-migration-banner a {
      color: #80a7e9;
    }

    /* Fallback for OS-level dark mode */
    @media (prefers-color-scheme: dark) {
      html:not([data-bs-theme="light"]):not(.light) .theme-banner-wrapper {
        background-color: var(--bs-body-bg, #20252b);
      }
      html:not([data-bs-theme="light"]):not(.light) .theme-migration-banner {
        background-color: #1a273b;
        color: #e6efff;
        box-shadow: 0 4px 6px rgba(0,0,0,0.3);
      }
      html:not([data-bs-theme="light"]):not(.light) .theme-migration-banner a {
        color: #80a7e9;
      }
    }
  `;
  document.head.appendChild(styleTag);

  // Create the Wrapper
  var wrapper = document.createElement('div');
  wrapper.id = 'migration-banner-wrapper';
  wrapper.className = 'theme-banner-wrapper';

  // Create the Banner
  var banner = document.createElement('div');
  banner.className = 'theme-migration-banner';
  banner.innerHTML = '⚠️ <strong>Archived Docs:</strong> Visit <a href="https://mcp-toolbox.dev/">mcp-toolbox.dev</a> for the latest version.';
  wrapper.appendChild(banner);

  // Inject the wrapper into the center information column
  var contentArea = document.querySelector('.td-content') || document.querySelector('main');
  if (contentArea) {
    contentArea.prepend(wrapper);
  } else {
    console.warn("Could not find the main content column to inject the banner.");
  }

  // Calculate navbar height synchronously to correctly offset the sticky wrapper
  var navbar = document.querySelector('.td-navbar');
  var navbarHeight = navbar ? navbar.offsetHeight : 64;
  wrapper.style.top = navbarHeight + 'px';
});
