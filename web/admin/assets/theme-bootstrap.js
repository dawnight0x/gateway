try {
  document.documentElement.dataset.theme = localStorage.getItem('gateway.theme') || 'light';
} catch {
  document.documentElement.dataset.theme = 'light';
}
