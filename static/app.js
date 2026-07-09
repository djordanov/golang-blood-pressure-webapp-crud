function setDefaultDateTime() {
  let now = new Date();
  document.querySelector('#bpo-new-date').value
    = `${now.getFullYear()}-${(now.getMonth()+1).toString().padStart(2, "0")}-${now.getDate().toString().padStart(2, "0")}`;
  document.querySelector('#bpo-new-time').value
    = `${now.getHours().toString().padStart(2, "0")}:${now.getMinutes().toString().padStart(2, "0")}`;
}

// Close modal when clicking outside (on the backdrop)
document.addEventListener('click', (event) => {
  const modal = document.querySelector('#add-new-bpo-dialog');
  if (modal && modal.open && event.target === modal) {
    modal.close();
  }
});

function resetNewBpoModal(triggerElement) {
  triggerElement.closest('form').reset();
  triggerElement.closest('dialog').close();
}
