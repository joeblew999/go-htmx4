// Evaluates kit/i18n/intltest cases with JavaScript's Intl: the one implementation behind the workerd oracle
// (oracle.mjs) and the browser drift check (browser.go). No logic beyond dispatching each case to the Intl API it
// names. No imports, so it loads unchanged in workerd and in any browser.

// "@date:<ISO>" arguments become Date objects; other JSON values pass through.
function arg(a) {
  return typeof a === "string" && a.startsWith("@date:") ? new Date(a.slice(6)) : a;
}

function run(c) {
  if (c.ctor) {
    // Generic: new Intl[ctor](locale, options)[method](...args)[field]
    const obj = c.ctor === "Locale" ? new Intl.Locale(c.locale, c.options) : new Intl[c.ctor](c.locale, c.options);
    let out = obj[c.method](...(c.args ?? []).map(arg));
    if (c.field) out = out[c.field];
    return typeof out === "string" ? out : JSON.stringify(out);
  }
  switch (c.api) {
    case "NumberFormat":
      return new Intl.NumberFormat(c.locale, c.options).format(c.input);
    case "PluralRules": {
      const pr = new Intl.PluralRules(c.locale, c.options);
      return pr.select(Number(c.input));
    }
    case "PluralRules.selectRange":
      return new Intl.PluralRules(c.locale, c.options).selectRange(Number(c.input), Number(c.input2));
    case "Locale": {
      const loc = new Intl.Locale(c.locale);
      const nf = new Intl.NumberFormat(c.locale).resolvedOptions();
      const dtf = new Intl.DateTimeFormat(c.locale, { hour: "numeric", timeZone: "UTC" }).resolvedOptions();
      return JSON.stringify({
        maximize: loc.maximize().toString(),
        minimize: loc.minimize().toString(),
        resolved: nf.locale,
        numberingSystem: nf.numberingSystem,
        hourCycle: dtf.hourCycle,
      });
    }
  }
  throw new Error("unknown api " + c.api);
}

// evaluate returns {meta, results: {id: output}}; a case that throws records "ERR: <error name>".
export function evaluate(cases) {
  const results = {};
  for (const c of cases) {
    try {
      results[c.id] = run(c);
    } catch (e) {
      results[c.id] = "ERR: " + e.name;
    }
  }
  const meta = {
    userAgent: navigator.userAgent,
    intlDefault: new Intl.NumberFormat().resolvedOptions().locale,
  };
  return { meta, results };
}
