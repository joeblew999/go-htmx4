// Local-only workerd Worker: evaluates kit/i18n/intltest cases with JavaScript's Intl and returns
// {meta, results: {id: output}}. No logic beyond dispatching each case to the Intl API it names.
import cases from "./cases.json";

function run(c) {
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

export default {
  async fetch() {
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
    return Response.json({ meta, results });
  },
};
