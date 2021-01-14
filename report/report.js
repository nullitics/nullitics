let today = new Date();
let oldest = new Date(fullData.Start);
today.setHours(0, 0, 0, 0);
oldest.setHours(0, 0, 0, 0);

let start, end;

const HOUR = 60 * 60 * 1000;
const DAY = 24 * HOUR;

const zeros = n => Array(n).fill(0);
const extend = (a, from, n) => zeros(n).map((_, i) => a[from + i] || 0);
const sum = a => a.reduce((acc, i) => (acc + i) | 0, 0);
const framify = ({Rows}, from, n) =>
  Rows.map(({Name, Values}) => [Name, ...extend(Values, from, n)]);
const total = m => sum(m.map(a => sum(a)));

const slice = (start, end, key) => {
  start.setHours(0, 0, 0, 0);
  end.setHours(0, 0, 0, 0);
  let buckets = Math.max((end - start) / DAY, 0);
  let from = 0;
  let source = fullData;
  let fmt = new Intl.DateTimeFormat([], {dateStyle: 'short'});
  let increment = DAY;
  if (buckets <= 1 && end.getTime() === today.getTime()) {
    buckets = 24;
    source = dailyData;
    fmt = new Intl.DateTimeFormat([], {timeStyle: 'short'});
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

const render = (c, el) => (el.innerHTML = c);

const last = n => {
  start = new Date(today);
  end = new Date(today);
  start.setDate(start.getDate() - n);
  render(App(), nullitics);
};

const Nav = () => `
  <nav style="margin-bottom:3rem;">
    <a href="#" onclick="last(1)">Today</a> |
    <a href="#" onclick="last(7)">Last 7 days</a> |
    <a href="#" onclick="last(30)">Last 30 days</a>
  </nav>
`;

const rank = key => {
  const [items] = slice(start, end, key);
  const reduced = items
    .map(([k, ...v]) => [k, sum(v)])
    .sort(([x, a], [y, b]) => b - a);
  const total = sum(reduced.map(([k, v]) => v));
  return [reduced, total];
};

const Panel = (
  title,
  columns,
  rows,
  contents,
) => `<section style="grid-column:${columns};grid-row:${rows};padding:0 1rem 1rem 1rem;border:3px solid black;border-radius:8px;">
      <h2>${title}</h2>
      <div>${contents}</div>
    </section>`;

const Graph = () => {
  const sum = v => v.reduce((a, i) => a + i, 0);
  const [[sessions], labels] = slice(start, end, 'Sessions');
  const totalSessions = sum(sessions.slice(1));

  const [paths] = slice(start, end, 'URIs');
  const views = [];
  for (let i = 0; i < labels.length; i++) {
    let sum = 0;
    for (let j = 0; j < paths.length; j++) {
      sum = sum + paths[j][i+1];
    }
    views[i-1] = sum;
  }
  const max = views.slice(1).reduce((m, i) => Math.max(i, m), 0);
  const totalViews = sum(views);
  const bounceRate = Math.round(totalSessions / totalViews * 100);
  console.log(views);
  return `
    <h3>${totalSessions} visitors / ${totalViews} views / ${bounceRate}% bounce rate</h3>
    <div style="display:grid;height:100px;widht:100%;
      grid-template-columns:repeat(${labels.length},1fr);
      grid-template-rows:repeat(102,1fr);
      grid-column-gap:5px;">
      ${labels
        .map(
          (label, i) => `<div style="background:#ccc;
          grid-column:${i+1}/${i+1};
          grid-row-start:${(101 - (views[i] / max) * 100) | 0};
          grid-row-end:102;"></div>`,
        )
        .join('')}
      ${labels
        .map(
          (label, i) => `<div style="background:black;
          grid-column:${i+1}/${i+1};
          grid-row-start:${(101 - (sessions[i + 1] / max) * 100) | 0};
          grid-row-end:102;"></div>`,
        )
        .join('')}
  </div>`;
};

const List = (name, limit) => {
  const [list, total] = rank(name);
  const percent = (a, b) => (b === 0 ? 0 : Math.floor((100 * a) / b));
  return `<div style="display:grid;grid-template-columns:1fr 3rem 3rem;">
        ${
          list.length === 0
            ? '<p>No data</p>'
            : list
                .slice(0, limit)
                .map(
                  ([name, count]) => `
            <span style="text-overflow:ellipsis;white-space:nowrap;overflow:hidden;">${name}</span>
            <span>${count}</span>
            <span>${percent(count, total)}%</span>
        `,
                )
                .join('')
        }
      </div>
    `;
};

const Map = () => {
  const [countries, total] = rank('Countries');
  const percent = (a, b) => (b === 0 ? 0 : Math.floor((100 * a) / b));
  const svg = new DOMParser().parseFromString(worldMapSVG, 'image/svg+xml');
  setTimeout(() => {
    const max = countries.reduce((m, [cn, v]) => Math.max(m, v), 0);
    const worldmap = document.getElementById('worldmap');
    worldmap.innerHTML = worldMapSVG;
    worldmap
      .querySelector('svg')
      .setAttributeNS(null, 'fill', 'rgba(0,0,0,0.15)');
    console.log(countries);
    countries.map(([cn, v]) => {
      const el = worldmap.querySelector('#' + cn.toLowerCase());
      if (el) {
        el.setAttributeNS(
          null,
          'fill',
          `rgba(0, 0, 0, ${(0.8 * v) / max + 0.2})`,
        );
      }
    });
  }, 0);
  return `
        <div style="display:grid;grid-template-columns:3fr 1fr;grid-gap:2rem;">
          <div id="worldmap" style="margin:1rem;"></div>
          <div style="display:grid;grid-template-columns:1fr 3rem 3rem;">
          ${
            countries.length === 0
              ? '<p>No data</p>'
              : countries
                  .slice(0, 15)
                  .map(
                    ([name, count]) => `
              <span style="text-overflow:ellipsis;white-space:nowrap;overflow:hidden;">${name}</span>
              <span>${count}</span>
              <span>${percent(count, total)}%</span>
          `,
                  )
                  .join('')
          }
          </div>
        </div>
      `;
};

const Grid = () => `
  <div style="display:grid;grid-template-columns:repeat(auto-fit, minmax(360px, 1fr));grid-gap:2rem;">
    ${Panel('Sessions', '1/-1', '', Graph())}
    ${Panel('Paths', '1/2', 'span 2', List('URIs', 30))}
    ${Panel('Referrals', '-2/-1', 'span 1', List('Refs', 20))}
    ${Panel('Devices', '-2/-1', 'span 1', List('Devices'))}
    ${Panel('Map', '1/-1', '', Map())}
    ${Panel('Desktop', '', '', List('Devices'))}
    ${Panel('Mobile', '', '', List('Devices'))}
  </div>
`;

const App = () => `
  ${Nav()}
  ${Grid()}
`;

last(1);
