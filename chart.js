async function updateChart() {
    let symbol = document.getElementById('symbol').value;
    let resp = await fetch('/data?symbol=' + symbol);
    let data = await resp.json(); 
    let chart = document.getElementById('chart');
    Plotly.newPlot(chart, data.data, data.layout);
}

document.addEventListener('DOMContentLoaded', function () {
    document.getElementById('generate').onclick = updateChart;
});
