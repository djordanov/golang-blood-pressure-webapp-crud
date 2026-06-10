function setDefaultDateTime() {
  let now = new Date();
  document.querySelector('#bpo-new-date').value
    = `${now.getFullYear()}-${now.getMonth().toString().padStart(2, "0")}-${now.getDay().toString().padStart(2, "0")}`;
  document.querySelector('#bpo-new-time').value
    = `${now.getHours().toString().padStart(2, "0")}:${now.getMinutes().toString().padStart(2, "0")}`;
}

