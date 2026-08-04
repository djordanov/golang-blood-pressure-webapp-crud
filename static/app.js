function setDefaultDateTime() {
  let now = new Date();
  document.querySelector('#bpo-date').value
    = `${now.getFullYear()}-${(now.getMonth() + 1).toString().padStart(2, "0")}-${now.getDate().toString().padStart(2, "0")}`;
  document.querySelector('#bpo-time').value
    = `${now.getHours().toString().padStart(2, "0")}:${now.getMinutes().toString().padStart(2, "0")}`;
}

// Close modal when clicking outside (on the backdrop)
document.addEventListener('click', (event) => {
  const modalCRUD = document.querySelector('#bpo-CRUD-dialog');
  if (modalCRUD && modalCRUD.open && event.target === modalCRUD) {
    modalCRUD.close();
  }

  const modalImport = document.querySelector('#import-dialog');
  if (modalImport && modalImport.open && event.target === modalImport) {
    modalImport.close();
  }
});

function resetFormModal(triggerElement) {
  triggerElement.closest('form').reset();
  triggerElement.closest('dialog').close();
}
