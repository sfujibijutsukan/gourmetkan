(() => {
  function mergeFiles(existing, added) {
    const dt = new DataTransfer();
    for (const f of existing) {
      dt.items.add(f);
    }
    for (const f of added) {
      dt.items.add(f);
    }
    return dt.files;
  }

  function bindDropzones() {
    const dropzones = document.querySelectorAll('.js-dropzone');
    dropzones.forEach((zone) => {
      const input = zone.querySelector('input[type="file"]');
      if (!input) {
        return;
      }

      const activate = () => zone.classList.add('is-dragover');
      const deactivate = () => zone.classList.remove('is-dragover');

      ['dragenter', 'dragover'].forEach((eventName) => {
        zone.addEventListener(eventName, (event) => {
          event.preventDefault();
          event.stopPropagation();
          activate();
        });
      });

      ['dragleave', 'dragend', 'drop'].forEach((eventName) => {
        zone.addEventListener(eventName, (event) => {
          event.preventDefault();
          event.stopPropagation();
          deactivate();
        });
      });

      zone.addEventListener('drop', (event) => {
        const dropped = event.dataTransfer && event.dataTransfer.files ? Array.from(event.dataTransfer.files) : [];
        const imageFiles = dropped.filter((f) => /^image\//.test(f.type));
        if (imageFiles.length === 0) {
          return;
        }
        try {
          const currentFiles = Array.from(input.files || []);
          input.files = mergeFiles(currentFiles, imageFiles);
          input.dispatchEvent(new Event('change', { bubbles: true }));
        } catch (_err) {
          const dt = new DataTransfer();
          imageFiles.forEach((f) => dt.items.add(f));
          input.files = dt.files;
          input.dispatchEvent(new Event('change', { bubbles: true }));
        }
      });
    });
  }

  function bindPhotoRemoveButtons() {
    const buttons = document.querySelectorAll('.js-remove-photo-btn');
    buttons.forEach((button) => {
      button.addEventListener('click', () => {
        const card = button.closest('.edit-photo-item');
        if (!card) {
          return;
        }
        const checkbox = card.querySelector('input[name="remove_photos"]');
        if (!checkbox) {
          return;
        }
        checkbox.checked = !checkbox.checked;
        card.classList.toggle('is-marked-remove', checkbox.checked);
        button.setAttribute('aria-pressed', checkbox.checked ? 'true' : 'false');
        button.textContent = checkbox.checked ? '↻' : '×';
      });
    });
  }

  function bindBaseSelect() {
    const select = document.getElementById('base_id');
    if (!select || !select.form) {
      return;
    }
    const submitBaseSelect = () => {
      if (typeof select.form.requestSubmit === 'function') {
        select.form.requestSubmit();
        return;
      }
      select.form.submit();
    };
    select.addEventListener('change', submitBaseSelect);
    select.addEventListener('input', submitBaseSelect);
  }

  document.addEventListener('DOMContentLoaded', () => {
    bindBaseSelect();
    bindDropzones();
    bindPhotoRemoveButtons();
  });
})();
