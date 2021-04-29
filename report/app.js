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
  let buckets = Math.max(Math.ceil((end - start) / DAY), 0);
  let from = 0;
  let source = fullData;
  let fmt = new Intl.DateTimeFormat([], {day: '2-digit', month: 'short'});
  let increment = DAY;
  if (buckets <= 1 && end.getTime() === today.getTime()) {
    buckets = 24;
    source = dailyData;
    fmt = new Intl.DateTimeFormat([], {
      hour: '2-digit',
      hourCycle: 'h23',
      minute: '2-digit',
    });
    increment = HOUR;
  } else {
    from = Math.ceil((start - oldest) / DAY);
  }
  const labels = zeros(buckets).map((_, i) =>
    fmt.format(new Date(start.getTime() + increment * i)),
  );
  if (!source[key] || !source[key].Rows) {
    return [[], labels];
  }
  const frame = framify(source[key], from, buckets).filter(row => row.slice(1).some(x => x != 0));
  return [frame, labels];
};

const rank = key => {
  const [items] = slice(start, end, key);
  const reduced = items
    .map(([k, ...v]) => [k, sum(v)])
    .sort(([x, a], [y, b]) => b - a);
  const total = sum(reduced.map(([k, v]) => v));
  return [reduced, total];
};
let today = new Date();
let oldest = new Date(fullData.Start);
today.setHours(0, 0, 0, 0);
oldest.setHours(0, 0, 0, 0);

const sliceMap = (start, end, key) => {
  const [items] = slice(start, end, key);
  return items.reduce((m, [key, ...value]) => ({...m, [key]: value.reduce((a, n) => a+n, 0)}), {});
}

const render = () => {
  const {from, to} = document.querySelector('nu-date-range');
  document.querySelectorAll('[data-filter]').forEach(el => {
      const x = sliceMap(from, to, el.dataset.filter);
      el.items = sliceMap(from, to, el.dataset.filter);
  });
  const sum = v => v.reduce((a, i) => a + i, 0);
  const [paths, labels] = slice(from, to, 'URIs');
  const [[sessions = zeros(labels.length+1)]] = slice(from, to, 'Sessions');
  const views = labels.map((_, i) => paths.reduce((a, p) => a + p[i+1], 0));
  const totalSessions = sum(sessions.slice(1));
  const totalViews = sum(views);

  document.querySelector('nu-summary').visitors = totalSessions;
  document.querySelector('nu-summary').views = totalViews;
  document.querySelector('nu-graph').labels = labels;
  document.querySelector('nu-graph').points = [views, sessions.slice(1)];
};

window.onload = () => {
  render();
  window.cloak.classList.remove('hidden');
};
