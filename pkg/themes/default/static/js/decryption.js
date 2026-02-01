/**
 * Client-side AES-256-GCM Decryption for Private Posts
 *
 * Uses Web Crypto API with PBKDF2 key derivation matching the Go server-side encryption.
 *
 * Encryption format: base64(salt || nonce || ciphertext || tag)
 * - Salt: 16 bytes (for PBKDF2 key derivation)
 * - Nonce: 12 bytes (GCM IV)
 * - Ciphertext: variable length
 * - Tag: 16 bytes (GCM authentication tag, appended to ciphertext by GCM)
 *
 * UX Features:
 * - Explicit "Remember me" opt-in for session storage (secure by default)
 * - Detailed error messages for different failure modes
 * - Full keyboard accessibility with ARIA labels
 * - Multi-post unlock: one password unlocks all posts with the same encryption key
 */
(function() {
  'use strict';

  // Constants matching the Go encryption package
  const SALT_SIZE = 16;
  const NONCE_SIZE = 12;
  const PBKDF2_ITERATIONS = 100000;
  const KEY_SIZE = 32; // 256 bits

  // Error types for better UX
  // pragma: allowlist nextline secret
  const ErrorType = {
    EMPTY_PASSWORD: 'empty_password', // pragma: allowlist secret
    WRONG_PASSWORD: 'wrong_password', // pragma: allowlist secret
    CORRUPTED_DATA: 'corrupted_data',
    BROWSER_UNSUPPORTED: 'browser_unsupported',
    UNKNOWN: 'unknown'
  };

  // Error messages for each type
  // pragma: allowlist nextline secret
  const ErrorMessages = {
    [ErrorType.EMPTY_PASSWORD]: 'Please enter a password.', // pragma: allowlist secret
    [ErrorType.WRONG_PASSWORD]: 'Incorrect password. Please check your password and try again.', // pragma: allowlist secret
    [ErrorType.CORRUPTED_DATA]: 'The encrypted data appears to be corrupted or incomplete.',
    [ErrorType.BROWSER_UNSUPPORTED]: 'Your browser does not support the required encryption features. Please use a modern browser.',
    [ErrorType.UNKNOWN]: 'An unexpected error occurred. Please try again.'
  };

  /**
   * Derive a 256-bit key from password using PBKDF2 with SHA-256.
   * Must match the Go DeriveKey function parameters exactly.
   */
  async function deriveKey(password, salt) {
    const encoder = new TextEncoder();
    const passwordKey = await crypto.subtle.importKey(
      'raw',
      encoder.encode(password),
      'PBKDF2',
      false,
      ['deriveBits', 'deriveKey']
    );

    return crypto.subtle.deriveKey(
      {
        name: 'PBKDF2',
        salt: salt,
        iterations: PBKDF2_ITERATIONS,
        hash: 'SHA-256'
      },
      passwordKey,
      { name: 'AES-GCM', length: KEY_SIZE * 8 },
      false,
      ['decrypt']
    );
  }

  /**
   * Decrypt base64-encoded ciphertext using AES-256-GCM.
   * Input format: base64(salt || nonce || ciphertext)
   * Returns: { success: true, content: string } or { success: false, errorType: string }
   */
  async function decrypt(encryptedBase64, password) {
    // Check for empty password
    if (!password || password.trim() === '') {
      return { success: false, errorType: ErrorType.EMPTY_PASSWORD };
    }

    try {
      // Decode base64
      let combined;
      try {
        const binaryString = atob(encryptedBase64);
        combined = new Uint8Array(binaryString.length);
        for (let i = 0; i < binaryString.length; i++) {
          combined[i] = binaryString.charCodeAt(i);
        }
      } catch (e) {
        return { success: false, errorType: ErrorType.CORRUPTED_DATA };
      }

      // Validate minimum size (salt + nonce + at least 1 byte + GCM tag)
      const MIN_SIZE = SALT_SIZE + NONCE_SIZE + 1 + 16;
      if (combined.length < MIN_SIZE) {
        return { success: false, errorType: ErrorType.CORRUPTED_DATA };
      }

      // Extract salt, nonce, and ciphertext
      const salt = combined.slice(0, SALT_SIZE);
      const nonce = combined.slice(SALT_SIZE, SALT_SIZE + NONCE_SIZE);
      const ciphertext = combined.slice(SALT_SIZE + NONCE_SIZE);

      // Derive key from password
      const key = await deriveKey(password, salt);

      // Decrypt using AES-GCM
      try {
        const decrypted = await crypto.subtle.decrypt(
          {
            name: 'AES-GCM',
            iv: nonce
          },
          key,
          ciphertext
        );

        // Convert to string
        const decoder = new TextDecoder('utf-8');
        return { success: true, content: decoder.decode(decrypted) };
      } catch (decryptError) {
        // GCM authentication failure typically means wrong password
        // or tampered data
        if (decryptError.name === 'OperationError') {
          return { success: false, errorType: ErrorType.WRONG_PASSWORD };
        }
        throw decryptError;
      }
    } catch (error) {
      console.error('Decryption error:', error);
      return { success: false, errorType: ErrorType.UNKNOWN };
    }
  }

  /**
   * Generate a storage key for a given encryption key name.
   */
  function getStorageKey(keyName) {
    return 'markata_decrypt_' + (keyName || 'default');
  }

  /**
   * Get saved password from session storage if user opted in.
   */
  function getSavedPassword(keyName) {
    try {
      return sessionStorage.getItem(getStorageKey(keyName));
    } catch (e) {
      return null;
    }
  }

  /**
   * Save password to session storage (only if user opted in).
   */
  function savePassword(keyName, password) {
    try {
      sessionStorage.setItem(getStorageKey(keyName), password);
    } catch (e) {
      // Session storage might be disabled
    }
  }

  /**
   * Clear saved password from session storage.
   */
  function clearSavedPassword(keyName) {
    try {
      sessionStorage.removeItem(getStorageKey(keyName));
    } catch (e) {
      // Ignore
    }
  }

  /**
   * Announce message to screen readers.
   */
  function announceToScreenReader(message) {
    const announcement = document.createElement('div');
    announcement.setAttribute('role', 'status');
    announcement.setAttribute('aria-live', 'polite');
    announcement.setAttribute('aria-atomic', 'true');
    announcement.className = 'sr-only';
    announcement.textContent = message;
    document.body.appendChild(announcement);

    setTimeout(function() {
      document.body.removeChild(announcement);
    }, 1000);
  }

  /**
   * Handle decryption for a single encrypted content block.
   */
  function handleDecryption(container, allContainers) {
    const encryptedData = container.dataset.encrypted;
    const keyName = container.dataset.keyName || 'default';
    const input = container.querySelector('.encrypted-content__input');
    const button = container.querySelector('.encrypted-content__button');
    const errorEl = container.querySelector('.encrypted-content__error');
    const lockedEl = container.querySelector('.encrypted-content__locked');
    const decryptedEl = container.querySelector('.encrypted-content__decrypted');
    const rememberCheckbox = container.querySelector('.encrypted-content__remember');

    if (!encryptedData || !input || !button || !lockedEl || !decryptedEl) {
      console.error('Encrypted content: missing required elements');
      return;
    }

    // Track decryption state
    let isDecrypting = false;

    async function attemptDecryption(passwordOverride) {
      if (isDecrypting) return;

      const password = passwordOverride || input.value.trim();
      if (!password) {
        showError(ErrorType.EMPTY_PASSWORD);
        input.focus();
        return;
      }

      // Set decrypting state
      isDecrypting = true;
      input.disabled = true;
      button.disabled = true;
      button.setAttribute('aria-busy', 'true');
      const originalButtonText = button.textContent;
      button.textContent = 'Decrypting...';
      hideError();

      const result = await decrypt(encryptedData, password);

      if (result.success) {
        // Show decrypted content
        decryptedEl.innerHTML = result.content;
        decryptedEl.style.display = 'block';
        lockedEl.style.display = 'none';

        // Update ARIA attributes
        container.setAttribute('aria-label', 'Decrypted content');
        decryptedEl.setAttribute('tabindex', '-1');
        decryptedEl.focus();

        // Announce success to screen readers
        announceToScreenReader('Content decrypted successfully');

        // Save password if user opted in
        if (rememberCheckbox && rememberCheckbox.checked) {
          savePassword(keyName, password);
        }

        // Try to unlock other containers with the same key
        unlockOtherContainers(keyName, password, container, allContainers);
      } else {
        showError(result.errorType);
        input.disabled = false;
        button.disabled = false;
        button.setAttribute('aria-busy', 'false');
        button.textContent = originalButtonText;
        input.focus();
        input.select();

        // If using saved password and it failed, clear it
        if (passwordOverride) {
          clearSavedPassword(keyName);
        }
      }

      isDecrypting = false;
    }

    function showError(errorType) {
      if (errorEl) {
        const message = ErrorMessages[errorType] || ErrorMessages[ErrorType.UNKNOWN];
        errorEl.textContent = message;
        errorEl.style.display = 'block';
        errorEl.setAttribute('role', 'alert');
        errorEl.setAttribute('aria-live', 'assertive');
      }
    }

    function hideError() {
      if (errorEl) {
        errorEl.style.display = 'none';
        errorEl.removeAttribute('role');
        errorEl.removeAttribute('aria-live');
      }
    }

    // Event listeners
    button.addEventListener('click', function() {
      attemptDecryption();
    });

    input.addEventListener('keydown', function(e) {
      if (e.key === 'Enter') {
        e.preventDefault();
        attemptDecryption();
      }
    });

    // Clear error on input
    input.addEventListener('input', function() {
      hideError();
    });

    // Try to auto-decrypt from session storage
    const savedPassword = getSavedPassword(keyName);
    if (savedPassword) {
      // Pre-fill and check the remember checkbox
      input.value = savedPassword;
      if (rememberCheckbox) {
        rememberCheckbox.checked = true;
      }
      attemptDecryption(savedPassword);
    } else {
      // Focus the input field
      input.focus();
    }
  }

  /**
   * Try to unlock other encrypted content blocks with the same key.
   */
  async function unlockOtherContainers(keyName, password, currentContainer, allContainers) {
    for (const container of allContainers) {
      // Skip the container we just unlocked
      if (container === currentContainer) continue;

      // Skip already decrypted containers
      const lockedEl = container.querySelector('.encrypted-content__locked');
      if (!lockedEl || lockedEl.style.display === 'none') continue;

      // Check if same key
      const containerKeyName = container.dataset.keyName || 'default';
      if (containerKeyName !== keyName) continue;

      // Try to decrypt
      const encryptedData = container.dataset.encrypted;
      const decryptedEl = container.querySelector('.encrypted-content__decrypted');

      if (!encryptedData || !decryptedEl) continue;

      const result = await decrypt(encryptedData, password);
      if (result.success) {
        decryptedEl.innerHTML = result.content;
        decryptedEl.style.display = 'block';
        lockedEl.style.display = 'none';
        container.setAttribute('aria-label', 'Decrypted content');
      }
    }
  }

  /**
   * Initialize all encrypted content blocks on the page.
   */
  function init() {
    // Check if Web Crypto API is available
    if (!crypto || !crypto.subtle) {
      console.error('Web Crypto API is not available. Content cannot be decrypted.');
      document.querySelectorAll('.encrypted-content').forEach(function(container) {
        const errorEl = container.querySelector('.encrypted-content__error');
        if (errorEl) {
          errorEl.textContent = ErrorMessages[ErrorType.BROWSER_UNSUPPORTED];
          errorEl.style.display = 'block';
          errorEl.setAttribute('role', 'alert');
        }
        // Disable the form
        const input = container.querySelector('.encrypted-content__input');
        const button = container.querySelector('.encrypted-content__button');
        if (input) input.disabled = true;
        if (button) button.disabled = true;
      });
      return;
    }

    // Find all encrypted content blocks
    const allContainers = Array.from(document.querySelectorAll('.encrypted-content[data-encrypted]'));

    // Initialize each one, passing reference to all containers for multi-unlock
    allContainers.forEach(function(container) {
      handleDecryption(container, allContainers);
    });
  }

  // Initialize when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
