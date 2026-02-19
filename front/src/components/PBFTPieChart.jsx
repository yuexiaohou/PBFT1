import { PieChart } from '@mui/x-charts/PieChart';
export default function PBFTPieChart({ validators }) {
    const valueMap = { commit: 0, precommit: 0, reject: 0 };
    validators.forEach(v => valueMap[v.vote]++);
    const data = [
        { label: 'Commit', value: valueMap.commit },
        { label: 'Precommit', value: valueMap.precommit },
        { label: 'Reject', value: valueMap.reject },
    ];
    return <PieChart data={data} width={400} height={200} />;
}