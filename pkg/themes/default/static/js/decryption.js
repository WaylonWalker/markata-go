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
 */
(function() {
  'use strict';

  // Constants matching the Go encryption package
  const SALT_SIZE = 16;
  const NONCE_SIZE = 12;
  const PBKDF2_ITERATIONS = 100000;
  const KEY_SIZE = 32; // 256 bits

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
   */
  async function decrypt(encryptedBase64, password) {
    // Decode base64
    const binaryString = atob(encryptedBase64);
    const combined = new Uint8Array(binaryString.length);
    for (let i = 0; i < binaryString.length; i++) {
      combined[i] = binaryString.charCodeAt(i);
    }

    // Extract salt, nonce, and ciphertext
    const salt = combined.slice(0, SALT_SIZE);
    const nonce = combined.slice(SALT_SIZE, SALT_SIZE + NONCE_SIZE);
    const ciphertext = combined.slice(SALT_SIZE + NONCE_SIZE);

    // Derive key from password
    const key = await deriveKey(password, salt);

    // Decrypt using AES-GCM
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
    return decoder.decode(decrypted);
  }

  /**
   * Handle decryption for a single encrypted content block.
   */
  async function handleDecryption(container) {
    const encryptedData = container.dataset.encrypted;
    const input = container.querySelector('.encrypted-content__input');
    const button = container.querySelector('.encrypted-content__button');
    const errorEl = container.querySelector('.encrypted-content__error');
    const lockedEl = container.querySelector('.encrypted-content__locked');
    const decryptedEl = container.querySelector('.encrypted-content__decrypted');

    if (!encryptedData || !input || !button || !lockedEl || !decryptedEl) {
      console.error('Encrypted content: missing required elements');
      return;
    }

    async function attemptDecryption() {
      const password = input.value.trim();
      if (!password) {
        showError('Please enter a password');
        return;
      }

      // Disable input while decrypting
      input.disabled = true;
      button.disabled = true;
      button.textContent = 'Decrypting...';
      hideError();

      try {
        const decryptedContent = await decrypt(encryptedData, password);

        // Show decrypted content
        decryptedEl.innerHTML = decryptedContent;
        decryptedEl.style.display = 'block';
        lockedEl.style.display = 'none';

        // Store in session storage so user doesn't have to re-enter
        const key = 'encrypted_' + hashString(encryptedData.substring(0, 50));
        try {
          sessionStorage.setItem(key, password);
        } catch (e) {
          // Session storage might be disabled
        }
      } catch (error) {
        console.error('Decryption failed:', error);
        showError('Decryption failed. Please check your password and try again.');
        input.disabled = false;
        button.disabled = false;
        button.textContent = 'Decrypt';
        input.focus();
        input.select();
      }
    }

    function showError(message) {
      if (errorEl) {
        errorEl.textContent = message;
        errorEl.style.display = 'block';
      }
    }

    function hideError() {
      if (errorEl) {
        errorEl.style.display = 'none';
      }
    }

    // Event listeners
    button.addEventListener('click', attemptDecryption);
    input.addEventListener('keypress', function(e) {
      if (e.key === 'Enter') {
        attemptDecryption();
      }
    });

    // Try to auto-decrypt from session storage
    const key = 'encrypted_' + hashString(encryptedData.substring(0, 50));
    try {
      const savedPassword = sessionStorage.getItem(key);
      if (savedPassword) {
        input.value = savedPassword;
        attemptDecryption();
      }
    } catch (e) {
      // Session storage might be disabled
    }

    // Focus the input field
    input.focus();
  }

  /**
   * Simple hash function for generating session storage keys.
   */
  function hashString(str) {
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
      const char = str.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash; // Convert to 32-bit integer
    }
    return Math.abs(hash).toString(36);
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
          errorEl.textContent = 'Your browser does not support the required encryption APIs.';
          errorEl.style.display = 'block';
        }
      });
      return;
    }

    // Find and initialize all encrypted content blocks
    document.querySelectorAll('.encrypted-content[data-encrypted]').forEach(handleDecryption);
  }

  // Initialize when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
