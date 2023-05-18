import { lazy, useEffect, useState } from "react";

import { Loading, Select } from "../shared";
import { useWebSocket } from "../../lib/hooks/use-web-socket";

const FileBrowser = lazy(() => import("./FileBrowser"));

const LOCAL_STORAGE_KEY = "nitric-local-dash-storage-history";

const StorageExplorer = () => {
  const [selectedBucket, setSelectedBucket] = useState<string>();
  const { data, loading } = useWebSocket();

  const { buckets } = data || {};

  useEffect(() => {
    if (buckets?.length && !selectedBucket) {
      const previousBucket = localStorage.getItem(
        `${LOCAL_STORAGE_KEY}-last-bucket`
      );

      setSelectedBucket(
        buckets.find((b) => b === previousBucket) || buckets[0]
      );
    }
  }, [buckets]);

  useEffect(() => {
    if (selectedBucket) {
      // set history
      localStorage.setItem(`${LOCAL_STORAGE_KEY}-last-bucket`, selectedBucket);
    }
  }, [selectedBucket]);

  return (
    <Loading delay={400} conditionToShow={!loading}>
      {selectedBucket ? (
        <div className="flex max-w-7xl flex-col md:flex-row gap-8 md:pr-8">
          <div className="w-full flex flex-col gap-8">
            <h2 className="text-2xl font-medium text-blue-800">
              Bucket - {selectedBucket}
            </h2>
            <div>
              <nav className="flex items-end gap-4" aria-label="Breadcrumb">
                <ol className="flex min-w-[200px] items-center gap-4">
                  <li className="w-full">
                    <Select
                      id="bucket-select"
                      items={buckets || []}
                      label="Select Bucket"
                      selected={selectedBucket}
                      setSelected={setSelectedBucket}
                      display={(v) => (
                        <div className="flex items-center p-0.5 text-lg gap-4">
                          {v}
                        </div>
                      )}
                    />
                  </li>
                </ol>
              </nav>
            </div>
            <div className="bg-white shadow sm:rounded-lg">
              <div className="px-4 py-5 sm:p-6 flex flex-col gap-4">
                <FileBrowser bucket={selectedBucket} />
              </div>
            </div>
          </div>
        </div>
      ) : !buckets?.length ? (
        <div>
          <p>
            Please refer to our documentation on{" "}
            <a
              className="underline"
              target="_blank"
              href="https://nitric.io/docs/storage#buckets"
              rel="noreferrer"
            >
              creating buckets
            </a>{" "}
            as we are unable to find any existing buckets.
          </p>
          <p>
            To ensure that the buckets are created, execute an API that utilizes
            them.
          </p>
        </div>
      ) : null}
    </Loading>
  );
};

export default StorageExplorer;
