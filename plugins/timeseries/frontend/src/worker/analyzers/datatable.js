async function analyze(qd) {
  if (
    qd.keys.length > 6 ||
    !qd.dataset_array.every((ds) => ds.length < 50000 && ds.length > 0)
  ) {
    return {}; // Don't display table for huge datasets.
  }
  let cols = qd.dataset_array.map((data, i) => {
    // If the dataset has only one datapoint that is an object, tell datatable to run a pre-transform,
    // to display the keys as rows instead of columns
    if (data.length == 1 && typeof data[0].d === "object" && Object.keys(data[0].d).length > 1) {
      return {
        label: qd.keys[i],
        transform: "expand",
        columns: [{ prop: 'd_k', name: "Key" }, { prop: 'd_v', name: "Value" }]
      };
    }

    // Otherwise, show one row per datapoint
    let columns = [{ prop: "t", name: "Timestamp" }];
    if (data.some((dp) => dp.dt !== undefined)) {
      columns.push({
        prop: "dt",
        name: "Duration",
      });
    }

    if (typeof data[0].d !== "object") {
      columns.push({ prop: "d", name: "Data" });
    } else {
      // It is an object, so find the properties, and make them table headers rather than just the raw data
      let headers = {};
      let isWeird = false;
      data.forEach((dp) => {
        if (typeof dp.d !== "object") {
          isWeird = true;
        } else {
          Object.keys(dp.d).forEach((k) => {
            headers[k] = true;
          });
        }
      });

      if (isWeird) {
        // Just give the raw data, since wtf
        columns.push({ prop: "d_", name: "Data" });
      } else {
        Object.keys(headers).forEach((k) => {
          columns.push({ prop: "d_" + k, name: k });
        });
      }
    }
    return { columns, label: qd.keys[i] };
  });

  return {
    datatable: {
      weight: 20,
      title: "Data Table",
      visualization: "datatable",
      config: cols,
    },
  };
}

export default analyze;
