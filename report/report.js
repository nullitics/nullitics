let today = new Date();
let oldest = new Date(fullData.Start);
today.setHours(0, 0, 0, 0);
oldest.setHours(0, 0, 0, 0);

let start, end;

const HOUR = 60 * 60 * 1000;
const DAY = 24 * HOUR;

// Zeros returns an array of N elements filled with zeros.
const zeros = n => Array(n).fill(0);
// Sum returns the sum of all elements in array A, or zero if it is empty.
const sum = a => a.reduce((acc, i) => (acc + i) | 0, 0);
// Total returns the sum of all elements in a matrix M (array of arrays).
const total = m => sum(m.map(a => sum(a)));
const extend = (a, from, n) => zeros(n).map((_, i) => a[from + i] || 0);
const framify = ({Rows}, from, n) =>
  Rows.map(({Name, Values}) => [Name, ...extend(Values, from, n)]);

const slice = (start, end, key) => {
  start.setHours(0, 0, 0, 0);
  end.setHours(0, 0, 0, 0);
  let buckets = Math.max((end - start) / DAY, 0);
  let from = 0;
  let source = fullData;
  let fmt = new Intl.DateTimeFormat([], {day: 'numeric', month: 'short'});
  let increment = DAY;
  if (buckets <= 1 && end.getTime() === today.getTime()) {
    buckets = 24;
    source = dailyData;
    fmt = new Intl.DateTimeFormat([], {hour: '2-digit', hourCycle: 'h23', minute:'2-digit'});
    increment = HOUR;
  } else {
    from = (start - oldest) / DAY;
  }
  const labels = zeros(buckets).map((_, i) =>
    fmt.format(new Date(start.getTime() + increment * i)),
  );
  if (!source[key] || !source[key].Rows) {
    return [[], labels];
  }
  const frame = framify(source[key], from, buckets);
  return [frame, labels];
};

const last = n => {
  start = new Date(today);
  end = new Date(today);
  start.setDate(start.getDate() - n);
  document.querySelector('.sessions .data').innerHTML = Graph();
  document.querySelector('.paths .data').innerHTML = List('URIs', 15);
  document.querySelector('.refs .data').innerHTML = List('Refs', 15);
  document.querySelector('.countries .data').innerHTML = WorldMap();
  document.querySelector('.devices .data').innerHTML = List('Devices');
};

const rank = key => {
  const [items] = slice(start, end, key);
  const reduced = items
    .map(([k, ...v]) => [k, sum(v)])
    .sort(([x, a], [y, b]) => b - a);
  const total = sum(reduced.map(([k, v]) => v));
  return [reduced, total];
};

const Graph = () => {
  const sum = v => v.reduce((a, i) => a + i, 0);
  const [[sessions = []], labels] = slice(start, end, 'Sessions');
  const totalSessions = sum(sessions.slice(1));

  const [paths] = slice(start, end, 'URIs');
  const views = [];
  for (let i = 0; i < labels.length; i++) {
    let sum = 0;
    for (let j = 0; j < paths.length; j++) {
      sum = sum + paths[j][i + 1];
    }
    views[i] = sum;
  }
  const maxViews = views.slice(1).reduce((m, i) => Math.max(i, m), 0);
  const max = [5,10,25,50,100].find(n => maxViews < n) || Math.ceil(maxViews/50)*50;
  const totalViews = sum(views);
  const bounceRate = totalViews ? Math.round((totalSessions / totalViews) * 100) : 0;
  document.querySelector('.sessions .visitors span').innerText = totalSessions;
  document.querySelector('.sessions .views span').innerText = totalViews;
  document.querySelector('.sessions .bounce-rate span').innerText = bounceRate;

  return `
    <div style="display:grid;height:250px;widht:100%;
      grid-template-columns:repeat(${labels.length},1fr);
      grid-template-rows:repeat(252,1fr);
      grid-column-gap:10px;">
      <div style="border-bottom:1px solid var(--color-background-grey);grid-row:200;grid-column:1/-1;"></div>
      <div style="border-bottom:1px solid var(--color-background-grey);grid-row:150;grid-column:1/-1;"></div>
      <div style="border-bottom:1px solid var(--color-background-grey);grid-row:100;grid-column:1/-1;"></div>
      <div style="border-bottom:1px solid var(--color-background-grey);grid-row:50;grid-column:1/-1;"></div>
      ${labels
        .map(
          (label, i) => `<div title="${views[i]} views"
          style="background:var(--color-accent);
          grid-column:${i + 1}/${i + 1};
          grid-row-start:${(251 - (views[i] / max) * 250) | 0};
          grid-row-end:252;"></div>`,
        )
        .join('')}
      ${labels
        .map(
          (label, i) => `<div title="${sessions[i+1]} visitors"
          style="background:var(--color-text);
          grid-column:${i + 1}/${i + 1};
          grid-row-start:${(251 - (sessions[i + 1] / max) * 250) | 0};
          grid-row-end:252;"></div>`,
        )
        .join('')}
      ${labels
        .map((label, i) => `<div style="font-size:10px;line-height:25px;place-self:center;grid-row:253;color:var(--color-text-light);grid-column:${i+1}">${label}</div>`).join('')}
  </div>`;
};

const List = (name, limit) => {
  const [list, total] = rank(name);
  const percent = (a, b) => (b === 0 ? 0 : Math.floor((100 * a) / b));
  if (list.length === 0) {
    return `<p>No data</p>`;
  }
  const listItem = ([name, count]) => `
    <span class="record">${name}</span>
    <span class="count">${count}</span>
    <span class="percent">${percent(count, total)}%</span>
    <span class="bar"><span style="width:${Math.max(
      1,
      percent(count, total),
    )}%"></span></span>
  `;
  return list
    .slice(0, limit)
    .map(listItem)
    .join('');
};

const WorldMap = () => {
  const [countries, total] = rank('Countries');
  const percent = (a, b) => (b === 0 ? 0 : Math.floor((100 * a) / b));
  const svg = new DOMParser().parseFromString(worldMapSVG, 'image/svg+xml');
  setTimeout(() => {
    const max = countries.reduce((m, [cn, v]) => Math.max(m, v), 0);
    const worldmap = document.getElementById('worldmap');
    worldmap.innerHTML = worldMapSVG;
    worldmap
      .querySelector('svg')
      .setAttributeNS(null, 'fill', 'var(--color-background-map)');
    countries.map(([cn, v]) => {
      const el = worldmap.querySelector('#' + cn.toLowerCase());
      if (el) {
        el.setAttributeNS(null, 'fill', 'var(--color-map)');
        el.setAttributeNS(null, 'opacity', `${(0.8 * v) / max + 0.2}`);
      }
    });
  }, 0);
  return `
        <div style="display:grid;grid-template-columns:3fr 1fr;grid-gap:2rem;align-items:center;">
          <div id="worldmap" style="margin:40px;"></div>
          <section class="list">
            <div class="data">
          ${
            countries.length === 0
              ? '<p>No data</p>'
              : countries
                  .slice(0, 15)
                  .map(
                    ([name, count]) => `
              <span class="record">${name}</span>
              <span class="count">${count}</span>
              <span class="percent">${percent(count, total)}%</span>
              <span class="bar"><span style="width:${Math.max(
                1,
                percent(count, total),
              )}%"></span></span>
          `,
                  )
                  .join('')
          }
            </div>
          </section>
        </div>
      `;
};

last(1);
window.cloak.style.display = 'block';
