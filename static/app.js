function setDefaultDateTime() {
  let now = new Date();
  document.querySelector('#bpo-new-date').value
    = `${now.getFullYear()}-${now.getMonth().toString().padStart(2, "0")}-${now.getDay().toString().padStart(2, "0")}`;
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

